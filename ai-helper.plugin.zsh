# shellcheck disable=2148,SC2168,SC1090,SC2125
local FOUND_AI_HELPER=$+commands[ai-helper]

if [[ $FOUND_AI_HELPER -eq 1 ]]; then
  echo "found aihelper"
  source <(ai-helper -completion zsh)
fi
