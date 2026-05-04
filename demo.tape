Output razify-demo.gif

Set FontSize 14
Set Width 1000
Set Height 550
Set Theme "Dracula"
Set TypingSpeed 50ms
Set Margin 20
Set BorderRadius 8
Set WindowBar Colorful

Hide
Type "clear"
Enter
Sleep 500ms
Show

Type "./razify --help"
Sleep 300ms
Enter
Sleep 2s

Type "./razify scan demo.env"
Sleep 300ms
Enter
Sleep 3s

Type "./razify diff demo.env demo.env.example"
Sleep 300ms
Enter
Sleep 3s

Type "./razify audit demo.env demo.env.example"
Sleep 300ms
Enter
Sleep 4s

Sleep 1s