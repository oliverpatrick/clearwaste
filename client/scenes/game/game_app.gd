extends Node

const DefinitionRegistryScript = preload("res://core/content/definition_registry.gd")

@onready var login_screen: LoginScreen = $LoginScreen
@onready var world_stream: WorldStream = $WorldStream
@onready var camera: Camera3D = $Camera
@onready var auth_client: Node = get_node("/root/AuthClient")

var ticket := ""
var account_id := 0
var character_id := 0
var world_host := ""
var world_port := 0


func _ready() -> void:
	camera.clear_current(false)
	login_screen.submitted.connect(_on_login_submitted)
	auth_client.login_succeeded.connect(_on_login_succeeded)
	auth_client.login_failed.connect(_on_login_failed)


func _on_login_submitted(email: String, password: String) -> void:
	var base_url := str(ProjectSettings.get_setting("account/base_url", "http://127.0.0.1:8080"))
	auth_client.login(base_url, email, password)


func _on_login_failed(message: String) -> void:
	login_screen.show_error(message)


func _on_login_succeeded(response: Dictionary) -> void:
	ticket = str(response.ticket)
	account_id = int(response.accountId)
	character_id = int(response.characterId)
	world_host = str(response.world.host)
	world_port = int(response.world.port)
	login_screen.hide()
	login_screen.queue_free()
	_load_world()


func _load_world() -> void:
	var registry := DefinitionRegistryScript.new()
	if not registry.load_content():
		push_error("Failed to load content")
		return
	world_stream.configure(registry)
	if not world_stream.load_region("0:0:0"):
		push_error("Failed to load region 0:0:0")
		return
	camera.look_at(Vector3(32.0, 0.0, 32.0), Vector3.UP)
	camera.current = true
