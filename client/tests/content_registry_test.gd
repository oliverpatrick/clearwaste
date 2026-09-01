extends SceneTree

const Registry = preload("res://core/content/definition_registry.gd")


func _init() -> void:
	var registry = Registry.new()
	assert(registry.load_content())
	assert(registry.items.size() == 3)
	assert(registry.item_by_id(1).name == "Bronze Axe")
	assert(registry.mobs.size() == 2)
	assert(registry.mob_by_id(2).name == "Woman")
	assert(registry.regions.size() == 4)
	assert(registry.region_at(1, 0).regionX == 1)
	quit()
