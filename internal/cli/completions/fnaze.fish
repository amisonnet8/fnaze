# fish completion for fnaze.
#
# It has no list of commands. At every TAB it asks `fnaze __complete` what can
# come next, so it is never out of date with the fnaze that is installed, and
# it can offer the groups, languages and paths of the .fnaze/ the line is in.
#
# Install it with:  fnaze completion fish > ~/.config/fish/completions/fnaze.fish
# or try it with:   fnaze completion fish | source

function __fnaze_complete
    # The words before the one being typed, without "fnaze" and their quotes.
    set -l words (commandline -opc)
    set -e words[1]
    # The word being typed, without its quotes and backslashes, as --word: an
    # empty word would be dropped, and "--word=" never is. It is quoted so
    # that the argument is there even where $cur is a list with nothing in it.
    set -l cur (commandline -ct | string replace -ra "[\\\\'\"]" '')
    fnaze __complete "--word=$cur" -- $words 2>/dev/null
end

# Where a file is wanted: after -C, and for the path of set (a new one).
function __fnaze_wants_file
    set -l words (commandline -opc)
    set -e words[1]
    test (count $words) -ge 1; and test "$words[-1]" = -C; and return 0
    test "$words[1]" = set
end

# fish puts in no space after a candidate that ends in "/", and escapes what
# it puts in, so a pattern (src/*_test.go) reaches fnaze as it is.
complete -c fnaze -f -a '(__fnaze_complete)'
complete -c fnaze -F -n __fnaze_wants_file
