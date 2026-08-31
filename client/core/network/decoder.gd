class_name ProtocolDecoder
extends RefCounted

static func decode_frame(data: PackedByteArray):
	if data.size() < 4:
		return null
	var size := _u16(data, 2)
	if size > MAX_PAYLOAD or data.size() != 4 + size:
		return null
	return {"id": _u16(data, 0), "payload": data.slice(4)}

static func decode_message(id: int, payload: PackedByteArray):
	match id:
		CONNECT_ACK, ENTITY_DESPAWN:
			if payload.size() != 4:
				return null
			return {"entity": _u32(payload, 0)}
		REGION_LOAD, REGION_UNLOAD:
			if payload.size() != 5 or payload[4] > 3:
				return null
			return {"x": _s16(payload, 0), "z": _s16(payload, 2), "plane": payload[4]}
		ENTITY_SPAWN:
			if payload.size() < 16 or payload[13] > 3:
				return null
			var name_end := 16 + payload[14]
			var definition_end := name_end + payload[15]
			if definition_end != payload.size():
				return null
			return {"entity": _u32(payload, 0), "type": payload[4], "x": _s32(payload, 5), "z": _s32(payload, 9), "plane": payload[13], "name": payload.slice(16, name_end).get_string_from_utf8(), "definition_id": payload.slice(name_end, definition_end).get_string_from_utf8()}
		ENTITY_MOVE:
			if payload.size() != 13 or payload[12] > 3:
				return null
			return {"entity": _u32(payload, 0), "x": _s32(payload, 4), "z": _s32(payload, 8), "plane": payload[12]}
		CHAT_MESSAGE:
			if payload.size() < 5 or payload[4] > 160 or payload.size() != 5 + payload[4]:
				return null
			return {"sender": _u32(payload, 0), "text": payload.slice(5).get_string_from_utf8()}
		INVENTORY:
			if payload.is_empty() or payload[0] > 30 or payload.size() != 1 + payload[0] * 5:
				return null
			var slots: Array = []
			for index in range(payload[0]):
				var offset := 1 + index * 5
				if payload[offset] >= 30 or _u16(payload, offset + 3) == 0:
					return null
				slots.append({"slot": payload[offset], "item": _u16(payload, offset + 1), "quantity": _u16(payload, offset + 3)})
			return {"slots": slots}
		SKILL:
			if payload.size() != 9:
				return null
			return {"skill": payload[0], "xp": _u64(payload, 1)}
		GROUND_ITEM:
			if payload.size() != 8:
				return null
			var quantity := _u32(payload, 4)
			if quantity == 0:
				return null
			return {"entity": _u32(payload, 0), "quantity": quantity}
		SYSTEM_MESSAGE:
			if payload.is_empty() or payload[0] > 160 or payload.size() != 1 + payload[0]:
				return null
			return {"text": payload.slice(1).get_string_from_utf8()}
		RESOURCE_STATE:
			if payload.size() != 7:
				return null
			return {"entity": _u32(payload, 0), "state": payload[4], "remaining": _u16(payload, 5)}
		PLAYER_ACTION:
			if payload.size() != 5 or payload[4] > 4:
				return null
			return {"entity": _u32(payload, 0), "action": payload[4]}
		ENTITY_FACE:
			if payload.size() != 8:
				return null
			return {"entity": _u32(payload, 0), "target": _u32(payload, 4)}
		ENTITY_HEALTH:
			if payload.size() != 12:
				return null
			return {"entity": _u32(payload, 0), "hp": _u32(payload, 4), "maximum": _u32(payload, 8)}
		ENTITY_HIT:
			if payload.size() != 8:
				return null
			return {"target": _u32(payload, 0), "damage": _u32(payload, 4)}
		DIALOGUE:
			if payload.size() < 5 or payload[4] == 0 or payload.size() != 5 + payload[4]:
				return null
			var bytes := payload.slice(5)
			var text = _strict_utf8(bytes)
			if text == null:
				return null
			return {"speaker": _u32(payload, 0), "text": text}
		COMBAT_STYLE:
			if payload.size() != 1 or payload[0] > COMBAT_STYLE_DEFENSIVE:
				return null
			return {"style": payload[0]}
	return null