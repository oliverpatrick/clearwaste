extends Node

const DefinitionRegistryScript = preload("res://core/content/definition_registry.gd")
const PlayerRegistryScript = preload("res://scripts/game/entities/player/player_registry.gd")
const ObjectRegistryScript = preload("res://scripts/game/entities/objects/object_registry.gd")
const Protocol = preload("res://core/network/protocol.gd")
const WallScene = preload("res://scenes/assets/wall.tscn")
const PlayerCameraScript = preload("res://scenes/player/player_camera.gd")

@onready var login_screen: LoginScreen = $LoginScreen
@onready var world_stream: WorldStream = $WorldStream
@onready var camera: Camera3D = $Camera
@onready var auth_client: Node = get_node("/root/AuthClient")
@onready var game_network_client: Node = get_node("/root/GameNetworkClient")

var ticket := ""
var account_id := 0
var character_id := 0
var world_host := ""
var world_port := 0
var _orbit := Vector2.ZERO
var player_registry: Node = null


func _ready() -> void:
	camera.clear_current(false)
	login_screen.submitted.connect(_on_login_submitted)
	auth_client.login_succeeded.connect(_on_login_succeeded)
	auth_client.login_failed.connect(_on_login_failed)
	game_network_client.connected.connect(_on_world_connected)
	game_network_client.bootstrap_received.connect(_on_bootstrap_received)
	game_network_client.message_received.connect(_on_world_message)
	game_network_client.disconnected.connect(_on_world_disconnected)


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
	game_network_client.connect_to_world(world_host, world_port, ticket)

func _on_world_disconnected(reason: String) -> void:
	if login_screen != null:
		login_screen.show_error(reason)

func _on_world_connected() -> void:
	pass

func _on_world_message(id: int, message: Variant) -> void:
	if id == Protocol.POSITION_UPDATE and player_registry != null and message is Dictionary:
		var player = player_registry.players.get(int(message.entity))
		if player != null: player.snap_to_tile(int(message.x), int(message.z), int(message.plane))

func _on_bootstrap_received(bootstrap: Dictionary) -> void:
	login_screen.hide()
	login_screen.queue_free()
	_load_world(bootstrap)


func _load_world(_bootstrap: Dictionary) -> void:
	var registry := DefinitionRegistryScript.new()
	if not registry.load_content():
		push_error("Failed to load content")
		return
	world_stream.configure(registry)
	if world_stream.load_all_regions() == 0:
		push_error("Failed to load regions")
		return
	_build_house(registry)
	player_registry = PlayerRegistryScript.new()
	player_registry.configure(registry, int(_bootstrap.local_entity_id))
	player_registry.local_player_ready.connect(_on_local_player_ready)
	add_child(player_registry)
	var objects := ObjectRegistryScript.new()
	objects.configure(registry)
	add_child(objects)
	for snapshot: Dictionary in _bootstrap.entities:
		var message := snapshot.duplicate()
		message["entity"] = message.id
		message["type"] = message.kind
		if message.kind == 0 or message.kind == 1:
			player_registry.handle_message(Protocol.ENTITY_SPAWN, message)
		elif message.kind == 2:
			objects._spawn_object(message)
		else:
			message["item_id"] = message.definition_id
			objects._spawn_object(message)
	if camera.get_script() == null:
		camera.set_script(PlayerCameraScript)
	if player_registry.players.has(int(_bootstrap.local_entity_id)):
		_on_local_player_ready(player_registry.players[int(_bootstrap.local_entity_id)])

func _on_local_player_ready(player: Node3D) -> void:
	if camera.has_method("configure"):
		camera.configure(player)

func _build_house(registry) -> void:
	var region = registry.region_at(0, 0, 0)
	for wall: Dictionary in region.collision.get("walls", []):
		var node := WallScene.instantiate()
		node.position = Vector3(float(wall.x) + 0.5, 1.0, float(wall.y) + 0.5)
		if wall.edge in ["east", "west"]: node.rotation.y = PI / 2.0
		add_child(node)

func _unhandled_input(event: InputEvent) -> void:
	if not camera.current: return
	if event is InputEventMouseMotion and Input.is_mouse_button_pressed(MOUSE_BUTTON_MIDDLE):
		camera.rotate_y(-event.relative.x * 0.01)
		camera.rotation.x = clamp(camera.rotation.x - event.relative.y * 0.01, -1.4, -0.1)
	elif event is InputEventMouseButton and event.pressed and event.button_index in [MOUSE_BUTTON_WHEEL_UP, MOUSE_BUTTON_WHEEL_DOWN]:
		camera.position += camera.global_basis.z * (2.0 if event.button_index == MOUSE_BUTTON_WHEEL_UP else -2.0)
	elif event is InputEventKey and event.pressed and event.keycode in [KEY_LEFT, KEY_RIGHT, KEY_UP, KEY_DOWN]:
		var delta := Vector3((1 if event.keycode == KEY_RIGHT else -1 if event.keycode == KEY_LEFT else 0), 0, (1 if event.keycode == KEY_DOWN else -1 if event.keycode == KEY_UP else 0))
		camera.position += delta * 2.0
