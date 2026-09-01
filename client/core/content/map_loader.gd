class_name MapLoader
extends RefCounted

const REGION_SIZE := 64


static func load_region(path: String) -> Dictionary:
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return {}
	var source = JSON.parse_string(file.get_as_text())
	return normalize_region(source) if source is Dictionary else {}


static func normalize_region(source: Dictionary) -> Dictionary:
	var regions := {}
	var region_x := int(source.regionX)
	var region_z := int(source.regionY)
	for plane: Dictionary in source.planes:
		var plane_id := int(plane.plane)
		regions["%d:%d:%d" % [region_x, region_z, plane_id]] = {
			"x": region_x,
			"z": region_z,
			"plane": plane_id,
			"heights": _normalize_heights(region_x, region_z, plane.height),
		}
	return regions


static func _normalize_heights(region_x: int, region_z: int, source: Dictionary) -> Array:
	var default_height := float(source.get("default", 0.0))
	var noise := FastNoiseLite.new()
	noise.noise_type = FastNoiseLite.TYPE_PERLIN
	noise.seed = int(ProjectSettings.get_setting("terrain/noise_seed", 0))
	noise.frequency = float(ProjectSettings.get_setting("terrain/noise_frequency", 0.02))
	var amplitude := float(ProjectSettings.get_setting("terrain/noise_amplitude", 8.0))
	var heights: Array = []
	for z in range(REGION_SIZE + 1):
		var row: Array = []
		for x in range(REGION_SIZE + 1):
			var height := default_height
			if is_zero_approx(default_height):
				height = noise.get_noise_2d(region_x * REGION_SIZE + x, region_z * REGION_SIZE + z) * amplitude
			row.append(height)
		heights.append(row)
	for override: Dictionary in source.get("overrides", []):
		heights[int(override.y)][int(override.x)] += float(override.value)
	return heights
