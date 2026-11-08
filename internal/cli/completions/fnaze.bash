# bash completion for fnaze.
#
# It has no list of commands. At every TAB it asks `fnaze __complete` what can
# come next, so it is never out of date with the fnaze that is installed, and
# it can offer the groups, languages and paths of the .fnaze/ the line is in.
#
# Install it with:  fnaze completion bash > ~/.local/share/bash-completion/completions/fnaze
# or try it with:   source <(fnaze completion bash)

# _fnaze_unquote sets REPLY to a word as bash keeps it, without its quotes and
# backslashes: fnaze wants the word, not how it was typed.
_fnaze_unquote() {
    REPLY=${1//\\/}
    REPLY=${REPLY//\'/}
    REPLY=${REPLY//\"/}
}

_fnaze() {
    local value i
    local -a words
    COMPREPLY=()
    for ((i = 1; i < COMP_CWORD; i++)); do
        _fnaze_unquote "${COMP_WORDS[i]}"
        words+=("$REPLY")
    done
    # The word being typed goes as --word: an empty word would be dropped by
    # some shells, and "--word=" never is.
    _fnaze_unquote "${COMP_WORDS[COMP_CWORD]}"
    while IFS=$'\t' read -r value _; do
        # A directory goes on: no space after it.
        [[ $value == */ ]] && compopt -o nospace 2>/dev/null
        # A pattern (src/*_test.go), or a name with a space, is quoted so that
        # it reaches fnaze as it is.
        if [[ $value == *[][\ \*\?\'\"\\\$\&\;\|\<\>\(\)\!\{\}\#\`]* ]]; then
            value="'${value//\'/\'\\\'\'}'"
        fi
        COMPREPLY+=("$value")
    done < <(command fnaze __complete "--word=$REPLY" -- "${words[@]}" 2>/dev/null)
}

# -o default: where nothing is offered (-C, a new path for set), files are.
complete -o default -F _fnaze fnaze
