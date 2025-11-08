package command

type FieldSeparator string

type flags struct {
	FieldSeparator FieldSeparator
}

func (f FieldSeparator) Configure(flags *flags) {
	flags.FieldSeparator = f
}
