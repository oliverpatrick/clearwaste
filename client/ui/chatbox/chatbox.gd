class_name Chatbox
extends PanelContainer

signal nearby_submitted(text: String)
@onready var chat_history: RichTextLabel = $VBoxContainer/ChatHistory
@onready var player_name: Label = $VBoxContainer/HBoxContainer/MarginContainer/PlayerName
@onready var chat_input: LineEdit = $VBoxContainer/HBoxContainer/ChatInput

func _ready() -> void:
	chat_input.text_submitted.connect(_submit)

func add_message(text: String) -> void:
	chat_history.append_text(text + "\n")

func _submit(text: String) -> void:
	if not text.strip_edges().is_empty():
		nearby_submitted.emit(text)
	chat_input.clear()
