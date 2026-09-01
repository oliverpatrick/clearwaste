class_name DefinitionRegistry
extends RefCounted

const MapLoaderScript = preload("res://core/content/map_loader.gd")

var items: Dictionary = {}
var mobs: Dictionary = {}
var regions: Dictionary = {}


func load_content(root_path: String = "") -> bool:
	if root_path.is_empty():
		root_path = ProjectSettings.globalize_path("res://../content_data")
	items.clear()
	mobs.clear()
	regions.clear()
	return _load_definitions(root_path.path_join("items"), items) \
		and _load_definitions(root_path.path_join("mobs"), mobs) \
		and _load_initial_region(root_path.path_join("map/region_0_0.json"))


func item_by_id(id: int):
	return items.get(id)


func mob_by_id(id: int):
	return mobs.get(id)


func region_at(x: int, y: int, plane: int = 0):
	return regions.get("%d:%d:%d" % [x, y, plane])


func _load_definitions(path: String, registry: Dictionary) -> bool:
	var directory := DirAccess.open(path)
	if directory == null:
		return false
	for file_name in directory.get_files():
		if file_name.get_extension().to_lower() != "json":
			continue
		var definition = _read_json(path.path_join(file_name))
		if definition == null:
			return false
		registry[int(definition.id)] = definition
	for directory_name in directory.get_directories():
		if not _load_definitions(path.path_join(directory_name), registry):
			return false
	return true


func _load_initial_region(path: String) -> bool:
	var loaded := MapLoaderScript.load_region(path)
	regions.merge(loaded)
	return not loaded.is_empty()


func _read_json(path: String):
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return null
	var value = JSON.parse_string(file.get_as_text())
	return value if value is Dictionary else null
