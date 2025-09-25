package irc

const (
	CodeNotice         = "NOTICE"
	CodeNickChange     = "NICK"
	CodeJoin           = "JOIN"
	CodeInvite         = "INVITE"
	CodePrivateMessage = "PRIVMSG"
	CodeQuit           = "QUIT"
	CodeError          = "ERROR"
)

const (
	CodeWhoIsReply   = "311"
	CodeEndOfWho     = "315"
	CodeEndOfWhoIs   = "318"
	CodeWhoReply     = "352"
	CodeNamesReply   = "353"
	CodeEndOfNames   = "366"
	CodeEndOfMotd    = "376"
	CodeNickReserved = "432"
	CodeNickInUse    = "433"
	CodeBanned       = "474"
)

const (
	MessageServerShuttingDown = "server shutting down"
	MessageClosingLink        = "closing link"
	MessageNetSplit           = "*.net *.split"
)
