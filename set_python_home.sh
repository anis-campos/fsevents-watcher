function check_if_command_exists() {
  command -v "$1" >/dev/null
}

PYTHON_EXECUTABLE=$(command -v python)

if [ "$PYTHON_EXECUTABLE" == "/usr/bin/python" ]; then
  # macOS System python
  PYTHON_HOME=/System/Library/Frameworks/Python.framework/Versions/2.7
elif check_if_command_exists pyenv && [ "$PYTHON_EXECUTABLE" == "$(pyenv root)/shims/python" ]; then
  # pyenv
  PYTHON_HOME="$(pyenv root)/versions/$(pyenv version-name)"
else
  # homebrew python
  PYTHON_HOME=/usr/local
fi

# For testing, only show up by running the script ( not sourcing )
[ "$0" == "$BASH_SOURCE" ]  && echo $PYTHON_HOME

export PYTHON_HOME
