class_name WorldStream
extends Node3D

const RegionMeshBuilderScript = preload("uid://cmajy2ls4qmaf") # world/region_mesh_builder.gd
var bundle
var loaded: Dictionary = {}

func configure(content_bundle) -> void:
	bundle = content_bundle

func load_region(key: String) -> bool:
	if bundle == null or loaded.has(key) or not bundle.regions.has(key):
		return false
	var region: Dictionary = bundle.regions[key]
	if region.plane != 0:
		loaded[key] = null
		return true
	var instance := RegionMeshBuilderScript.build(region)
	if instance == null:
		return false
	add_child(instance)
	loaded[key] = instance
	return true

func load_all_regions() -> int:
	if bundle == null:
		return 0
	var count := 0
	for key in bundle.regions:
		if load_region(str(key)):
			count += 1
	return count

func unload_region(key: String) -> void:
	if not loaded.has(key):
		return
	var instance = loaded[key]
	if instance != null:
		instance.queue_free()
	loaded.erase(key)
