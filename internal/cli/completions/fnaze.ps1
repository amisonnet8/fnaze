# PowerShell completion for fnaze.
#
# It has no list of commands. At every TAB it asks `fnaze __complete` what can
# come next, so it is never out of date with the fnaze that is installed, and
# it can offer the groups, languages and paths of the .fnaze/ the line is in.
#
# Install it with:  fnaze completion powershell >> $PROFILE
# or try it with:   fnaze completion powershell | Out-String | Invoke-Expression
Register-ArgumentCompleter -Native -CommandName fnaze -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    # The words before the one being typed, without "fnaze": the elements of
    # the line that end before the cursor.
    $words = @($commandAst.CommandElements | Select-Object -Skip 1 |
        Where-Object { $_.Extent.EndOffset -lt $cursorPosition } |
        ForEach-Object {
            if ($_ -is [System.Management.Automation.Language.StringConstantExpressionAst]) { $_.Value } else { $_.Extent.Text }
        })
    # The word being typed, without its quotes, as --word: an empty word would
    # be dropped, and "--word=" never is.
    $word = $wordToComplete -replace '^[''"]|[''"]$', ''

    $exitCode = $global:LASTEXITCODE
    $lines = fnaze __complete "--word=$word" -- @words 2>$null
    $global:LASTEXITCODE = $exitCode

    foreach ($line in $lines) {
        if (-not $line) { continue }
        $value, $description = $line -split "`t", 2
        # A pattern (src/*_test.go), or a name with a space, is quoted so that
        # it reaches fnaze as it is.
        $text = $value
        if ($value -match '[\s*?\[\]''"`$&;|<>(){}@,#]') { $text = "'" + ($value -replace "'", "''") + "'" }
        # A completion result must have a description to show.
        if (-not $description) { $description = $value }
        [System.Management.Automation.CompletionResult]::new($text, $value, 'ParameterValue', $description)
    }
}
