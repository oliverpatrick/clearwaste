extends Node

const DefinitionRegistryScript = preload("res://core/content/definition_registry.gd")

@onready var world_stream: WorldStream = $WorldStream
@onready var camera: Camera3D = $Camera


func _ready() -> void:
	var registry := DefinitionRegistryScript.new()
	if not registry.load_content():
		push_error("Failed to load content")
		return
	world_stream.configure(registry)
	if not world_stream.load_region("0:0:0"):
		push_error("Failed to load region 0:0:0")
		return
	camera.look_at(Vector3(32.0, 0.0, 32.0), Vector3.UP)
