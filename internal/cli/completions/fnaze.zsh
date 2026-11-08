#compdef fnaze
# zsh completion for fnaze.
#
# It has no list of commands. At every TAB it asks `fnaze __complete` what can
# come next, so it is never out of date with the fnaze that is installed, and
# it can offer the groups, languages and paths of the .fnaze/ the line is in.
#
# Install it with:  fnaze completion zsh > "${fpath[1]}/_fnaze"
# or try it with:   source <(fnaze completion zsh)
_fnaze() {
    local line value item
    local -a dirs others
    # The words before the one being typed, and that one as --word, without
    # their quotes: an empty word would be dropped by some shells, and
    # "--word=" never is.
    for line in "${(@f)$(command fnaze __complete "--word=${(Q)words[CURRENT]}" -- "${(@Q)words[2,CURRENT-1]}" 2>/dev/null)}"; do
        [[ -n $line ]] || continue
        # "value<TAB>desc" is "value:desc" for _describe, which wants a colon
        # in the value written as \:.
        value=${line%%$'\t'*}
        item=${value//:/\\:}
        [[ $line == *$'\t'* ]] && item+=":${line#*$'\t'}"
        # A directory goes on: no space after it.
        if [[ $value == */ ]]; then
            dirs+=("$item")
        else
            others+=("$item")
        fi
    done
    if (( ${#dirs} + ${#others} == 0 )); then
        # Where nothing is offered (-C, a new path for set), files are.
        _files
        return
    fi
    # zsh quotes what it puts in, so a pattern (src/*_test.go) reaches fnaze
    # as it is.
    (( ${#dirs} )) && _describe -t candidates 'fnaze' dirs -S ''
    (( ${#others} )) && _describe -t candidates 'fnaze' others
    return 0
}

if [ "$funcstack[1]" = "_fnaze" ]; then
    _fnaze "$@"
else
    compdef _fnaze fnaze
fi
