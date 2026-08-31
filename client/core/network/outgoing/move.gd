
static func encode_move(x: int, z: int, plane: int, mode: int, sequence: int) -> PackedByteArray:
	if plane < 0 or plane > 3 or mode < 0 or mode > 1:
		return PackedByteArray()
	var payload := PackedByteArray()
	_put_u32(payload, x)
	_put_u32(payload, z)
	payload.append(plane)
	payload.append(mode)
	_put_u32(payload, sequence)
	return encode_frame(MOVE_REQUEST, payload)