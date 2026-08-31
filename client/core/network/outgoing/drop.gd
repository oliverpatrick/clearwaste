
static func encode_drop(slot: int, quantity: int, sequence: int) -> PackedByteArray:
	if slot < 0 or slot >= 30 or quantity <= 0 or quantity > 65535:
		return PackedByteArray()
	var payload := PackedByteArray([slot])
	_put_u16(payload, quantity)
	_put_u32(payload, sequence)
	return encode_frame(DROP_ITEM, payload)