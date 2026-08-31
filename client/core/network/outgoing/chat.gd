static func encode_chat(text: String) -> PackedByteArray:
	var bytes := text.to_utf8_buffer()
	if bytes.is_empty() or bytes.size() > 160:
		return PackedByteArray()
	var payload := PackedByteArray([bytes.size()])
	payload.append_array(bytes)
	return encode_frame(CHAT, payload)