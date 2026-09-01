class_name AssetRegistry
extends RefCounted

const ITEM_ASSETS := {
	1: preload("res://scenes/items/axes/bronze_axe.tscn"),
	2: preload("res://scenes/items/pickaxes/bronze_pickaxe.tscn"),
}
const NPC_ASSETS := {
	1: preload("res://scenes/mobs/human/male/man_01.tscn"),
	2: preload("res://scenes/mobs/human/female/woman_01.tscn"),
}
const OBJECT_ASSETS := {}
const APPEARANCE_ASSETS := {0: preload("res://scenes/player/player.tscn")}

static func item_scene(item_id: int):
	return ITEM_ASSETS.get(item_id)

static func npc_scene(npc_id: int):
	return NPC_ASSETS.get(npc_id)

static func object_scene(object_id: int):
	return OBJECT_ASSETS.get(object_id)

static func appearance_scene(appearance_id: int):
	return APPEARANCE_ASSETS.get(appearance_id)
