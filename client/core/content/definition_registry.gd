class_name DefinitionRegistry
extends RefCounted

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
		and _load_regions(root_path.path_join("map"))


func item_by_id(id: int):
	return items.get(id)


func mob_by_id(id: int):
	return mobs.get(id)


func region_at(x: int, y: int):
	return regions.get(Vector2i(x, y))


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


func _load_regions(path: String) -> bool:
	var directory := DirAccess.open(path)
	if directory == null:
		return false
	for file_name in directory.get_files():
		if file_name.get_extension().to_lower() != "json":
			continue
		var region = _read_json(path.path_join(file_name))
		if region == null:
			return false
		regions[Vector2i(int(region.regionX), int(region.regionY))] = region
	return true


func _read_json(path: String):
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return null
	var value = JSON.parse_string(file.get_as_text())
	return value if value is Dictionary else null
