//
/*
 * Copyright 2021 The Nakama Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import Foundation
import NIO
import Logging
import SwiftProtobuf
import Atomics

public final class Socket : SocketProtocol {
    public var isConnected: Bool {
        return socketAdapter.isConnected
    }
    
    public var isConnecting: Bool {
        return socketAdapter.isConnecting
    }
    
    public var onConnect: ConnectHandler?
    
    public var onDisconnect: DisconnectHandler?
    
    public var onError: SocketErrorHandler?
    
    public var onChannelMessage: ChannelMessageHandler?
    
    public var onChannelPresence: ChannelPresenceHandler?
    
    public var onMatchmakerMatched: MatchmakerMatchedHandler?
    
    public var onMatchData: MatchDataHandler?
    
    public var onMatchPresence: MatchPresenceHandler?
    
    public var onNotifications: NotificationsHandler?
    
    public var onStatusPresence: StatusPresenceHandler?
    
    public var onStreamPresence: StreamPresenceHandler?
    
    public var onStreamData: StreamDataHandler?
    
    public var onPartyReceived: PartyReceivedHandler?
    
    public var onPartyClosed: PartyCloseHandler?
    
    public var onPartyData: PartyDataHandler?
    
    public var onPartyJoin: PartyJoinRequestHandler?
    
    public var onPartyPresence: PartyPresenceHandler?
    
    public var onPartyMatchmakerTicket: PartyMatchmakerTicketHandler?
    
    public var onPartyLeader: PartyLeaderHandler?
    
    let collationCounter = ManagedAtomic<Int>(0)
    let eventLoopGroup: EventLoopGroup
    let logger: Logger?
    
    public let host: String
    public let port: Int
    public let ssl: Bool
    
    var socketAdapter: SocketAdapter
    var collatedPromises = [String:Any]()
    
    init(host: String, port: Int, ssl: Bool, eventLoopGroup: EventLoopGroup, socketAdapter: SocketAdapter?, logger: Logger?) {
        self.host = host
        self.port = port
        self.ssl = ssl
        self.eventLoopGroup = eventLoopGroup
        self.logger = logger
        
        if let socketAdapter {
            self.socketAdapter = socketAdapter
        } else {
            self.socketAdapter = SocketAdapter(logger: logger)
        }
        
        self.socketAdapter.onConnect = {
            self.onConnect?()
        }
        
        self.socketAdapter.onDisconnect = {
            self.onDisconnect?()
        }
        
        self.socketAdapter.onError = { error in
            self.onError?(error)
        }
        
        self.socketAdapter.onReceiveData = { data in
            self.onReceivedData(data: data)
        }
    }
    
    public func connect(session: Session, appearOnline: Bool? = nil) {
        var components = URLComponents()
        components.scheme = self.ssl ? "wss" : "ws"
        components.host = self.host
        components.port = self.port
        components.path = "/ws"
        components.queryItems = [
            URLQueryItem(name: "token", value: session.token),
            URLQueryItem(name: "format", value: "protobuf"),
        ]
        
        if let appearOnline {
            components.queryItems!.append(URLQueryItem(name: "status", value: appearOnline ? "true" : "false"))
        }
        
        self.socketAdapter.connect(url: components.url!)
    }
    
    public func disconnect() {
        self.socketAdapter.disconnect()
    }
    
    func send<T: Message>(env: inout Nakama_Realtime_Envelope) async throws -> T {
        let counter = String(collationCounter.wrappingIncrementThenLoad(ordering: .relaxed))
        let promise = self.eventLoopGroup.next().makePromise(of: T.self)
        env.cid = String(counter)
        self.collatedPromises[counter] = promise
        
        let binaryData = try env.serializedData()
        self.socketAdapter.send(data: binaryData)
        return try await promise.futureResult.get()
    }
    
    private func onReceivedData(data: Data) {
        let response: Nakama_Realtime_Envelope
        do {
            response = try Nakama_Realtime_Envelope(serializedData: data)
            if (response.cid.isEmpty) {
                switch response.message {
                case .error(let error):
                    self.onError?(NakamaRealtimeError(error: error))
                case .channelMessage(let channelMessage):
                    self.onChannelMessage?(channelMessage)
                case .channelPresenceEvent(let channelPresenceEvent):
                    self.onChannelPresence?(channelPresenceEvent)
                case .matchData(let matchData):
                    self.onMatchData?(matchData)
                case .matchPresenceEvent(let matchPresenceEvent):
                    self.onMatchPresence?(matchPresenceEvent)
                case .matchmakerMatched(let matchermakerMatched):
                    self.onMatchmakerMatched?(matchermakerMatched)
                case .notifications(let notifications):
                    self.onNotifications?(notifications)
                case .statusPresenceEvent(let statusPresenceEvent):
                    self.onStatusPresence?(statusPresenceEvent)
                case .streamData(let streamData):
                    self.onStreamData?(streamData)
                case .streamPresenceEvent(let streamPresenceEvent):
                    self.onStreamPresence?(streamPresenceEvent)
                case .party(let party):
                    self.onPartyReceived?(party.toParty())
                case .partyJoinRequest(let joinRequest):
                    self.onPartyJoin?(joinRequest)
                case .partyData(let data):
                    self.onPartyData?(data)
                case .partyPresenceEvent(let partyPresenceEvent):
                    self.onPartyPresence?(partyPresenceEvent)
                case .partyMatchmakerTicket(let ticket):
                    self.onPartyMatchmakerTicket?(ticket)
                case .partyClose(let close):
                    self.onPartyClosed?(close)
                case .partyLeader(let leader):
                    self.onPartyLeader?(leader)
                default:
                    self.logger?.error("Unrecognised incoming uncollated message from server: \(try! response.jsonString())")
                }
            } else {
                if let collatedPromise = self.collatedPromises[response.cid] {
                    switch response.message {
                    case .error(let error):
                        if let promise = collatedPromise as? EventLoopPromise<Any> {
                            promise.fail(NakamaRealtimeError(error: error))
                        } else if let promise = collatedPromise as? EventLoopPromise<Google_Protobuf_Empty> {
                            promise.fail(NakamaRealtimeError(error: error))
                        }
                    case .rpc(let rpc):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Api_Rpc>
                        promise.succeed(rpc)
                    case .channel(let channel):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_Channel>
                        promise.succeed(channel)
                    case .channelMessageAck(let channelMessageAck):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_ChannelMessageAck>
                        promise.succeed(channelMessageAck)
                    case .match(let match):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_Match>
                        promise.succeed(match)
                    case .matchmakerTicket(let matchmakerTicket):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_MatchmakerTicket>
                        promise.succeed(matchmakerTicket)
                    case .status(let status):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_Status>
                        promise.succeed(status)
                    case .party(let party):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_Party>
                        promise.succeed(party)
                    case .partyMatchmakerTicket(let ticket):
                        let promise = collatedPromise as! EventLoopPromise<Nakama_Realtime_PartyMatchmakerTicket>
                        promise.succeed(ticket)
                    default:
                        self.logger?.error("Unrecognised incoming collated message from server: \(try! response.jsonString())")
                        // Handle empty or nil response from server
                        if let promise = collatedPromise as? EventLoopPromise<SwiftProtobuf.Google_Protobuf_Empty> {
                            promise.succeed(SwiftProtobuf.Google_Protobuf_Empty())
                        }
                    }
                } else {
                    self.logger?.error("No matching promise for incoming collation ID: \(response.cid)")
                }
            }
        } catch {
            self.logger?.error("Failed to deserialise message with error: \(error)")
            self.onError?(error)
            return
        }
    }
    
    public func joinChat(target: String, type: Nakama_Realtime_ChannelJoin.TypeEnum, persistence: Bool? = false, hidden: Bool? = false) async throws -> NakamaChannel {
        var req = Nakama_Realtime_ChannelJoin()
        req.target = target
        req.type = Int32(type.rawValue)
        req.hidden = Google_Protobuf_BoolValue(hidden ?? false)
        req.persistence = Google_Protobuf_BoolValue(persistence ?? false)
        
        var env = Nakama_Realtime_Envelope()
        env.channelJoin = req
        
        let resp: Nakama_Realtime_Channel = try await self.send(env: &env)
        return NakamaChannel(from: resp)
    }
    
    public func leaveChat(channelId: String) async throws -> Void {
        var req = Nakama_Realtime_ChannelLeave()
        req.channelID = channelId
        
        var env = Nakama_Realtime_Envelope()
        env.channelLeave = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func removeChatMessage(channelId: String, messageId: String) async throws -> Nakama_Realtime_ChannelMessageAck {
        var req = Nakama_Realtime_ChannelMessageRemove()
        req.channelID = channelId
        req.messageID = messageId
        
        var env = Nakama_Realtime_Envelope()
        env.channelMessageRemove = req
        
        return try await self.send(env: &env)
    }
    
    public func writeChatMessage(channelId: String, content: String) async throws -> Nakama_Realtime_ChannelMessageAck {
        var req = Nakama_Realtime_ChannelMessageSend()
        req.channelID = channelId
        req.content = content
        var env = Nakama_Realtime_Envelope()
        env.channelMessageSend = req
        
        return try await self.send(env: &env)
    }
    
    public func updateChatMessage(channelId: String, messageId: String, content: String) async throws -> Nakama_Realtime_ChannelMessageAck {
        var req = Nakama_Realtime_ChannelMessageUpdate()
        req.channelID = channelId
        req.messageID = messageId
        req.content = content
        var env = Nakama_Realtime_Envelope()
        env.channelMessageUpdate = req
        
        return try await self.send(env: &env)
    }
    
    public func createMatch(name: String? = nil) async throws -> Nakama_Realtime_Match {
        var req = Nakama_Realtime_MatchCreate()
        if let name {
            req.name = name
        }
        var env = Nakama_Realtime_Envelope()
        env.matchCreate = req
        
        return try await self.send(env: &env)
    }
    
    public func joinMatch(matchId: String, metadata: [String : String]? = nil) async throws -> Nakama_Realtime_Match {
        var req = Nakama_Realtime_MatchJoin()
        req.matchID = matchId
        if let metadata {
            req.metadata = metadata
        }
        var env = Nakama_Realtime_Envelope()
        env.matchJoin = req
        
        return try await self.send(env: &env)
    }
    
    public func joinMatchToken(token: String) async throws -> Nakama_Realtime_Match {
        var req = Nakama_Realtime_MatchJoin()
        req.token = token
        var env = Nakama_Realtime_Envelope()
        env.matchJoin = req
        
        return try await self.send(env: &env)
    }
    
    public func leaveMatch(matchId: String) async throws -> Void {
        var req = Nakama_Realtime_MatchLeave()
        req.matchID = matchId
        
        var env = Nakama_Realtime_Envelope()
        env.matchLeave = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func addMatchmaker(query: String? = "*", minCount: Int? = 2, maxCount: Int? = 8, stringProperties: [String : String]? = nil, numericProperties: [String : Double]? = nil, countMultiple: Int? = nil) async throws -> Nakama_Realtime_MatchmakerTicket {
        var req = Nakama_Realtime_MatchmakerAdd()
        req.query = query ?? "*"
        req.minCount = Int32(minCount ?? 2)
        req.maxCount = Int32(maxCount ?? 8)
        if let stringProperties {
            req.stringProperties = stringProperties
        }
        if let numericProperties {
            req.numericProperties = numericProperties
        }
        if let countMultiple {
            req.countMultiple = countMultiple.pbInt32Value
        }
        
        var env = Nakama_Realtime_Envelope()
        env.matchmakerAdd = req
        
        return try await self.send(env: &env)
    }
    
    public func removeMatchmaker(ticket: String) async throws -> Void {
        var req = Nakama_Realtime_MatchmakerRemove()
        req.ticket = ticket
        
        var env = Nakama_Realtime_Envelope()
        env.matchmakerRemove = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func sendMatchData(matchId: String, opCode: Int, data: String, presences: [Nakama_Realtime_UserPresence]? = nil) async throws {
        guard let data = data.data(using: .utf8) else {
            throw NakamaRealtimeError(text: "Unable to convert string to Data")
        }
        
        var req = Nakama_Realtime_MatchDataSend()
        req.matchID = matchId
        req.opCode = Int64(opCode)
        req.data = data
        if let presences {
            req.presences = presences
        }
        
        var env = Nakama_Realtime_Envelope()
        env.matchDataSend = req
        
        let binaryData = try! env.serializedData()
        self.socketAdapter.send(data: binaryData)
    }
    
    public func sendMatchData(matchId: String, opCode: Int, data: Data, presences: [Nakama_Realtime_UserPresence]? = nil) async throws {
        var req = Nakama_Realtime_MatchDataSend()
        req.matchID = matchId
        req.opCode = Int64(opCode)
        req.data = data
        if let presences {
            req.presences = presences
        }
        
        var env = Nakama_Realtime_Envelope()
        env.matchDataSend = req
        
        let binaryData = try! env.serializedData()
        self.socketAdapter.send(data: binaryData)
    }
    
    public func rpc(id: String, payload: String?) async throws -> Nakama_Api_Rpc {
        var req = Nakama_Api_Rpc()
        req.id = id
        if payload != nil {
            req.payload = payload!
        }
        
        var env = Nakama_Realtime_Envelope()
        env.rpc = req
        
        return try await send(env: &env)
    }
    
    public func followUsers(userIds: [String]) async throws -> Nakama_Realtime_Status {
        return try await self.followUsers(userIds: userIds, usernames: nil)
    }
    
    public func followUsers(userIds: [String]?, usernames: [String]?) async throws -> Nakama_Realtime_Status {
        var req = Nakama_Realtime_StatusFollow()
        if let userIds {
            req.userIds = userIds
        }
        if let usernames {
            req.usernames = usernames
        }
        
        var env = Nakama_Realtime_Envelope()
        env.statusFollow = req
        
        return try await send(env: &env)
    }
    
    public func unfollowUsers(userIds: [String]) async throws -> Void {
        var req = Nakama_Realtime_StatusUnfollow()
        req.userIds = userIds
        
        var env = Nakama_Realtime_Envelope()
        env.statusUnfollow = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func updateStatus(status: String) async throws -> Void {
        var req = Nakama_Realtime_StatusUpdate()
        req.status = Google_Protobuf_StringValue(status)
            
        var env = Nakama_Realtime_Envelope()
        env.statusUpdate = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func createParty(open: Bool, maxSize: Int) async throws -> Nakama_Realtime_Party {
        var env = Nakama_Realtime_Envelope()
        env.partyCreate.open = open
        env.partyCreate.maxSize = Int32(maxSize)
        
        return try await self.send(env: &env)
    }
    
    public func joinParty(partyId: String) async throws {
        var env = Nakama_Realtime_Envelope()
        env.partyJoin.partyID = partyId
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func leaveParty(partyId: String) async throws {
        var env = Nakama_Realtime_Envelope()
        env.partyLeave.partyID = partyId
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func closeParty(partyId: String) async throws {
        var env = Nakama_Realtime_Envelope()
        env.partyClose.partyID = partyId
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func sendPartyData(partyId: String, opCode: Int, data: Data) async throws {
        var env = Nakama_Realtime_Envelope()
        env.partyDataSend.partyID = partyId
        env.partyDataSend.opCode = Int64(opCode)
        env.partyDataSend.data = data
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func sendPartyData(partyId: String, opCode: Int, data: String) async throws {
        guard let data = data.data(using: .utf8) else {
            throw NakamaRealtimeError(text: "Unable to convert string to Data")
        }
        
        var env = Nakama_Realtime_Envelope()
        env.partyDataSend.partyID = partyId
        env.partyDataSend.opCode = Int64(opCode)
        env.partyDataSend.data = data
    }
    
    public func acceptPartyMember(partyId: String, presence: UserPresence) async throws {
        var req = Nakama_Realtime_PartyAccept()
        req.partyID = partyId
        req.presence = presence.toApiUserPresence()
        var env = Nakama_Realtime_Envelope()
        env.partyAccept = req
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func removePartyMember(partyId: String, presence: UserPresence) async throws {
        var env = Nakama_Realtime_Envelope()
        env.partyRemove.partyID = partyId
        env.partyRemove.presence = presence.toApiUserPresence()
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func addMatchmakerParty(partyId: String, query: String, minCount: Int, maxCount: Int, stringProperties: [String:String]? = nil, numericProperties: [String:Double]? = nil, countMultiple: Int? = nil) async throws -> Nakama_Realtime_PartyMatchmakerTicket {
        var env = Nakama_Realtime_Envelope()
        var add = Nakama_Realtime_PartyMatchmakerAdd()
        add.partyID = partyId
        add.query = query
        add.minCount = Int32(minCount)
        add.maxCount = Int32(maxCount)
        if let stringProperties {
            add.stringProperties = stringProperties
        }
        if let numericProperties {
            add.numericProperties = numericProperties
        }
        if let countMultiple {
            add.countMultiple = countMultiple.pbInt32Value
        }
        
        env.partyMatchmakerAdd = add
        
        return try await self.send(env: &env)
    }
    
    public func removeMatchmakerParty(partyId: String, ticket: String) async throws -> Void {
        var env = Nakama_Realtime_Envelope()
        env.partyMatchmakerRemove.partyID = partyId
        env.partyMatchmakerRemove.ticket = ticket
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
    
    public func listPartyJoinRequests(partyId: String) async throws -> Nakama_Realtime_PartyJoinRequest {
        var env = Nakama_Realtime_Envelope()
        env.partyJoinRequestList.partyID = partyId
        
        return try await self.send(env: &env)
    }
    
    public func promotePartyMember(partyId: String, partyMember: UserPresence) async throws -> Void {
        var env = Nakama_Realtime_Envelope()
        env.partyPromote.partyID = partyId
        env.partyPromote.presence = partyMember.toApiUserPresence()
        
        let _: Google_Protobuf_Empty = try await self.send(env: &env)
    }
}
