Output hashy.gif
Set Theme "neobones_light

Require echo

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600

Type "du -sh ."
Sleep 500ms
Enter
Sleep 2000ms

Type "find . -type f | wc -l"
Sleep 500ms
Enter
Sleep 2000ms

Type "time hashy"
Sleep 500ms
Enter
Sleep 10s
