class_name LoginScreen
extends Control

signal submitted(email: String, password: String)

@onready var email: LineEdit = $VBoxContainer/LoginContainer/VBoxContainer/MarginContainer/VBoxContainer/EmailLineEdit
@onready var password: LineEdit = $VBoxContainer/LoginContainer/VBoxContainer/MarginContainer/VBoxContainer/PasswordLineEdit
@onready var login_button: Button = $VBoxContainer/LoginContainer/VBoxContainer/MarginContainer/VBoxContainer/LoginButton
@onready var status_label: Label = $StatusLabel

func _on_login_button_pressed() -> void:
	if email.text.strip_edges().is_empty() or password.text.is_empty():
		show_error("Enter your email and password")
		return
	set_busy(true)
	submitted.emit(email.text, password.text)
	
func set_busy(busy: bool) -> void:
	login_button.disabled = busy
	status_label.text = "Connecting..." if busy else ""

func show_error(message: String) -> void:
	login_button.disabled = false
	status_label.text = message

func _on_register_button_pressed() -> void:
	# TODO: Send the user to a url for the website
	pass # Replace with function body.
