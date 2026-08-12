package authortoday

func DecodeText(secret, text string) string {
	secretRunes := append(reverseRunes([]rune(secret)), []rune("@_@")...)
	if len(secretRunes) == 0 {
		return text
	}
	out := make([]rune, 0, len(text))
	for i, r := range []rune(text) {
		out = append(out, r^secretRunes[i%len(secretRunes)])
	}
	return string(out)
}

func reverseRunes(r []rune) []rune {
	out := make([]rune, len(r))
	for i, v := range r {
		out[len(r)-1-i] = v
	}
	return out
}
