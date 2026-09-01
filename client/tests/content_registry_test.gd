extends SceneTree

const Registry = preload("res://core/content/definition_registry.gd")
const MapLoaderScript = preload("res://core/content/map_loader.gd")
const RegionMeshBuilderScript = preload("res://scripts/game/world/region_mesh_builder.gd")


func _init() -> void:
	var registry = Registry.new()
	assert(registry.load_content())
	assert(registry.items.size() == 3)
	assert(registry.item_by_id(1).name == "Bronze Axe")
	assert(registry.mobs.size() == 2)
	assert(registry.mob_by_id(2).name == "Woman")
	assert(registry.regions.size() == 4)
	var region: Dictionary = registry.region_at(0, 0, 0)
	assert(region.heights.size() == 65)
	assert(region.heights[0].size() == 65)
	var mesh := RegionMeshBuilderScript.build(region)
	assert(mesh != null)
	mesh.free()

	var source := {
		"regionX": 0,
		"regionY": 0,
		"planes": [{
			"plane": 0,
			"height": {"default": 0, "overrides": []},
		}],
	}
	var base: Dictionary = MapLoaderScript.normalize_region(source)["0:0:0"]
	assert(not is_zero_approx(base.heights[20][8]))
	source.planes[0].height.overrides = [{"x": 8, "y": 20, "value": 3.5}]
	var raised: Dictionary = MapLoaderScript.normalize_region(source)["0:0:0"]
	assert(is_equal_approx(raised.heights[20][8], base.heights[20][8] + 3.5))
	quit()
