extends Node

const Protocol = preload("uid://bvppiqbq80y0l")
const PROTOCOL_VERSION := 1

signal connected
signal disconnected(reason: String)
signal message_received(id: int, message: Variant)
signal bootstrap_received(bootstrap: Dictionary)

var peer := StreamPeerTCP.new()
var receive_buffer := PackedByteArray()
var _ticket := ""
var _state := "idle"
var _was_transport_connected := false
var _disconnect_reported := false
var entity_id := 0

func connect_to_world(host: String, port: int, ticket: String) -> Error:
	close()
	_ticket = ticket
	_state = "connecting"
	_was_transport_connected = false
	_disconnect_reported = false
	entity_id = 0
	receive_buffer.clear()
	return peer.connect_to_host(host, port)

func _process(_delta: float) -> void:
	peer.poll()
	var status := peer.get_status()
	if status == StreamPeerTCP.STATUS_CONNECTED:
		_was_transport_connected = true
		if _state == "connecting":
			_state = "server_hello"
			peer.put_data(Protocol.encode_client_hello(PROTOCOL_VERSION))
		_read_available()
	elif status == StreamPeerTCP.STATUS_ERROR:
		_report_disconnect("Connection failed")
	elif status == StreamPeerTCP.STATUS_NONE and _was_transport_connected:
		_report_disconnect("Disconnected from world")

func _read_available() -> void:
	var available := peer.get_available_bytes()
	if available <= 0:
		return
	var result := peer.get_data(available)
	if result[0] == OK:
		receive_buffer.append_array(result[1])
		_drain_frames()

func send_frame(frame: PackedByteArray) -> Error:
	if peer.get_status() != StreamPeerTCP.STATUS_CONNECTED or frame.is_empty():
		return ERR_UNAVAILABLE
	return peer.put_data(frame)

func close() -> void:
	peer.disconnect_from_host()

func _report_disconnect(reason: String) -> void:
	if _disconnect_reported:
		return
	_disconnect_reported = true
	_state = "idle"
	disconnected.emit(reason)

func _drain_frames() -> void:
	while receive_buffer.size() >= Protocol.HEADER_SIZE:
		var payload_size := Protocol._u32(receive_buffer, 2)
		if payload_size > Protocol.MAX_PAYLOAD:
			close()
			_report_disconnect("Invalid server frame")
			return
		var frame_size := Protocol.HEADER_SIZE + payload_size
		if receive_buffer.size() < frame_size:
			return
		var frame = Protocol.decode_frame(receive_buffer.slice(0, frame_size))
		receive_buffer = receive_buffer.slice(frame_size)
		if frame == null:
			_report_disconnect("Invalid server frame")
			return
		_handle_frame(frame.id, frame.payload)

func _handle_frame(id: int, payload: PackedByteArray) -> void:
	if _state == "server_hello":
		if id != Protocol.SERVER_HELLO:
			_report_disconnect("Invalid world handshake")
			return
		var hello = Protocol.decode_server_hello(payload)
		if hello == null or not hello.accepted or hello.version != PROTOCOL_VERSION:
			_report_disconnect("World protocol rejected")
			return
		_state = "login"
		peer.put_data(Protocol.encode_login_request(_ticket))
		return
	if _state == "login":
		if id == Protocol.LOGIN_ACCEPTED and payload.is_empty():
			_state = "connected"
			connected.emit()
			return
		if id == Protocol.LOGIN_REJECTED and payload.is_empty():
			_report_disconnect("World login rejected")
			return
		_report_disconnect("Invalid world login response")
		return
	if id == Protocol.WORLD_BOOTSTRAP:
		var bootstrap = Protocol.decode_bootstrap(payload)
		if bootstrap == null:
			_report_disconnect("Invalid world bootstrap")
		else:
			bootstrap_received.emit(bootstrap)
		return
	if id == Protocol.POSITION_UPDATE:
		if payload.size() != 17: _report_disconnect("Invalid position update")
		else: message_received.emit(id, {"entity": Protocol._u64(payload, 0), "x": Protocol._s32(payload, 8), "z": Protocol._s32(payload, 12), "plane": payload[16]})
		return
	var message = Protocol.decode_message(id, payload)
	message_received.emit(id, message)
