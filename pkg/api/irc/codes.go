package irc

const (
	CodeNotice         = "NOTICE"
	CodeJoin           = "JOIN"
	CodeInvite         = "INVITE"
	CodePrivateMessage = "PRIVMSG"
	CodeQuit           = "QUIT"
	CodeError          = "ERROR"
)

const (
	CodeNamesReply = "353"
	CodeEndOfNames = "366"
	CodeWhoReply   = "352"
	CodeEndOfWho   = "315"
	CodeWhoIsReply = "311"
	CodeEndOfWhoIs = "318"
	CodeBanned     = "474"
	CodeNickInUse  = "433"
)

const (
	MessageServerShuttingDown = "server shutting down"
	MessageClosingLink        = "closing link"
	MessageNetSplit           = "*.net *.split"
)
