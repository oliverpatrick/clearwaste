extends SceneTree

const Registry = preload("res://core/content/definition_registry.gd")
const MapLoaderScript = preload("res://core/content/map_loader.gd")
const RegionMeshBuilderScript = preload("res://scripts/game/world/region_mesh_builder.gd")
const AuthClientScript = preload("res://autoloads/auth_client.gd")
const GameScene = preload("res://scenes/game/game.tscn")


func _init() -> void:
	var registry = Registry.new()
	assert(registry.load_content())
	assert(registry.items.size() == 3)
	assert(registry.item_by_id(1).name == "Bronze Axe")
	assert(registry.mobs.size() == 2)
	assert(registry.mob_by_id(2).name == "Woman")
	assert(registry.regions.size() == 16)
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

	var valid_response := {
		"ticket": "opaque-value",
		"accountId": 41,
		"characterId": 73,
		"world": {"host": "127.0.0.1", "port": 7777},
	}
	assert(AuthClientScript.is_valid_login_response(valid_response))
	var invalid_response := valid_response.duplicate(true)
	invalid_response.ticket = ""
	assert(not AuthClientScript.is_valid_login_response(invalid_response))
	invalid_response = valid_response.duplicate(true)
	invalid_response.erase("world")
	assert(not AuthClientScript.is_valid_login_response(invalid_response))

	await process_frame
	assert(root.get_node_or_null("AuthClient") != null)
	assert(root.get_node_or_null("GameNetworkClient") != null)

	var game := GameScene.instantiate()
	root.add_child(game)
	await process_frame
	var login_screen := game.find_child("LoginScreen", true, false)
	assert(login_screen != null and login_screen.visible)
	assert(game.find_child("Region_0_0_0", true, false) == null)
	assert(root.get_camera_3d() == null)

	game._on_login_succeeded(valid_response)
	await process_frame
	game._on_world_connected()
	await process_frame
	assert(game.find_child("LoginScreen", true, false) == null)
	assert(game.find_child("Region_0_0_0", true, false) != null)
	assert(root.get_camera_3d() != null)
	game.free()
	quit()
