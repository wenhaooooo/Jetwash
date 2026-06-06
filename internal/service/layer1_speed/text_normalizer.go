package layer1_speed

import (
	"regexp"
	"strings"
	"unicode"
)

var fullTraditionalToSimplified = []struct{ trad, simp rune }{
	{'萬', '万'}, {'與', '与'}, {'專', '专'}, {'業', '业'}, {'東', '东'}, {'絲', '丝'}, {'丟', '丢'}, {'兩', '两'}, {'嚴', '严'}, {'喪', '丧'},
	{'個', '个'}, {'豐', '丰'}, {'醫', '医'}, {'區', '区'}, {'廠', '厂'}, {'產', '产'}, {'際', '际'}, {'禮', '礼'}, {'潤', '润'}, {'蘇', '苏'},
	{'藝', '艺'}, {'憶', '忆'}, {'嚴', '严'}, {'遷', '迁'}, {'邁', '迈'}, {'縮', '缩'}, {'牆', '墙'}, {'畫', '画'}, {'環', '环'}, {'濕', '湿'},
	{'廟', '庙'}, {'關', '关'}, {'嶺', '岭'}, {'麗', '丽'}, {'齊', '齐'}, {'辭', '辞'}, {'齒', '齿'}, {'證', '证'}, {'際', '际'}, {'臟', '脏'},
	{'髮', '发'}, {'鬧', '闹'}, {'關', '关'}, {'顏', '颜'}, {'錄', '录'}, {'夢', '梦'}, {'辦', '办'}, {'錄', '录'}, {'觀', '观'}, {'黃', '黄'},
	{'龍', '龙'}, {'驢', '驴'}, {'驚', '惊'}, {'驛', '驿'}, {'驗', '验'}, {'顛', '颠'}, {'顯', '显'}, {'鳳', '凤'}, {'鵝', '鹅'}, {'麵', '面'},
	{'麥', '麦'}, {'黨', '党'}, {'齋', '斋'}, {'鹽', '盐'}, {'藝', '艺'}, {'醫', '医'}, {'館', '馆'}, {'齊', '齐'}, {'劍', '剑'}, {'騎', '骑'},
	{'騷', '骚'}, {'驫', '骉'}, {'鬥', '斗'}, {'鬱', '郁'}, {'馮', '冯'}, {'鳴', '鸣'}, {'鵬', '鹏'}, {'鷄', '鸡'}, {'鷗', '鸥'}, {'鷹', '鹰'},
	{'鷂', '隼'}, {'鶴', '鹤'}, {'愛', '爱'}, {'國', '国'}, {'語', '语'}, {'學', '学'}, {'開', '开'}, {'門', '门'}, {'長', '长'}, {'車', '车'},
	{'書', '书'}, {'飛', '飞'}, {'聽', '听'}, {'說', '说'}, {'買', '买'}, {'賣', '卖'}, {'實', '实'}, {'認', '认'}, {'識', '识'}, {'記', '记'},
	{'計', '计'}, {'設', '设'}, {'論', '论'}, {'選', '选'}, {'擇', '择'}, {'決', '决'}, {'義', '义'}, {'務', '务'}, {'戰', '战'}, {'爭', '争'},
	{'勝', '胜'}, {'敗', '败'}, {'殺', '杀'}, {'傷', '伤'}, {'害', '害'}, {'槍', '枪'}, {'彈', '弹'}, {'炸', '炸'}, {'藥', '药'}, {'毒', '毒'},
	{'品', '品'}, {'販', '贩'}, {'賭', '赌'}, {'博', '博'}, {'黃', '黄'}, {'色', '色'}, {'暴', '暴'}, {'力', '力'}, {'詐', '诈'}, {'騙', '骗'},
	{'盜', '盗'}, {'竊', '窃'}, {'搶', '抢'}, {'劫', '劫'}, {'綁', '绑'}, {'架', '架'}, {'勒', '勒'}, {'索', '索'}, {'黑', '黑'}, {'社', '社'},
	{'會', '会'}, {'幫', '帮'}, {'派', '派'}, {'組', '组'}, {'織', '织'}, {'團', '团'}, {'體', '体'}, {'機', '机'}, {'構', '构'}, {'單', '单'},
	{'位', '位'}, {'個', '个'}, {'們', '们'}, {'它', '它'},
}

type TextNormalizer struct {
	traditionalToSimplified map[rune]rune
	simplifiedToTraditional map[rune]rune
	halfwidthToFullwidth    map[rune]rune
	fullwidthToHalfwidth    map[rune]rune

	interleavedCharsRegex *regexp.Regexp
	emojiRegex            *regexp.Regexp
	zeroWidthRegex        *regexp.Regexp
}

func NewTextNormalizer() *TextNormalizer {
	tn := &TextNormalizer{
		traditionalToSimplified: make(map[rune]rune),
		simplifiedToTraditional: make(map[rune]rune),
		halfwidthToFullwidth:    make(map[rune]rune),
		fullwidthToHalfwidth:    make(map[rune]rune),
	}

	tn.initMappings()
	tn.initRegexPatterns()

	return tn
}

func (tn *TextNormalizer) initMappings() {
	for _, pair := range fullTraditionalToSimplified {
		tn.traditionalToSimplified[pair.trad] = pair.simp
		tn.simplifiedToTraditional[pair.simp] = pair.trad
	}

	halfwidthFullwidthPairs := []struct {
		half, full rune
	}{
		{' ', '　'}, {'!', '！'}, {'"', '\uFF02'}, {'#', '＃'},
		{'$', '＄'}, {'%', '％'}, {'&', '＆'}, {'\'', '\uFF07'},
		{'(', '（'}, {')', '）'}, {'*', '＊'}, {'+', '＋'},
		{',', '，'}, {'-', '－'}, {'.', '．'}, {'/', '／'},
		{'0', '０'}, {'1', '１'}, {'2', '２'}, {'3', '３'},
		{'4', '４'}, {'5', '５'}, {'6', '６'}, {'7', '７'},
		{'8', '８'}, {'9', '９'}, {':', '：'}, {';', '；'},
		{'<', '＜'}, {'=', '＝'}, {'>', '＞'}, {'?', '？'},
		{'@', '＠'}, {'A', 'Ａ'}, {'B', 'Ｂ'}, {'C', 'Ｃ'},
		{'D', 'Ｄ'}, {'E', 'Ｅ'}, {'F', 'Ｆ'}, {'G', 'Ｇ'},
		{'H', 'Ｈ'}, {'I', 'Ｉ'}, {'J', 'Ｊ'}, {'K', 'Ｋ'},
		{'L', 'Ｌ'}, {'M', 'Ｍ'}, {'N', 'Ｎ'}, {'O', 'Ｏ'},
		{'P', 'Ｐ'}, {'Q', 'Ｑ'}, {'R', 'Ｒ'}, {'S', 'Ｓ'},
		{'T', 'Ｔ'}, {'U', 'Ｕ'}, {'V', 'Ｖ'}, {'W', 'Ｗ'},
		{'X', 'Ｘ'}, {'Y', 'Ｙ'}, {'Z', 'Ｚ'}, {'[', '［'},
		{'\\', '＼'}, {']', '］'}, {'^', '＾'}, {'_', '＿'},
		{'`', '｀'}, {'a', 'ａ'}, {'b', 'ｂ'}, {'c', 'ｃ'},
		{'d', 'ｄ'}, {'e', 'ｅ'}, {'f', 'ｆ'}, {'g', 'ｇ'},
		{'h', 'ｈ'}, {'i', 'ｉ'}, {'j', 'ｊ'}, {'k', 'ｋ'},
		{'l', 'ｌ'}, {'m', 'ｍ'}, {'n', 'ｎ'}, {'o', 'ｏ'},
		{'p', 'ｐ'}, {'q', 'ｑ'}, {'r', 'ｒ'}, {'s', 'ｓ'},
		{'t', 'ｔ'}, {'u', 'ｕ'}, {'v', 'ｖ'}, {'w', 'ｗ'},
		{'x', 'ｘ'}, {'y', 'ｙ'}, {'z', 'ｚ'}, {'{', '｛'},
		{'|', '｜'}, {'}', '｝'}, {'~', '～'},
	}

	for _, pair := range halfwidthFullwidthPairs {
		tn.halfwidthToFullwidth[pair.half] = pair.full
		tn.fullwidthToHalfwidth[pair.full] = pair.half
	}
}

func (tn *TextNormalizer) initRegexPatterns() {
	tn.interleavedCharsRegex = regexp.MustCompile(`[*_=<>~·⋅⋆☆★○●◇◆□■△▲▽▼◁◀▷▶◉◎░▒▓█▄▀褐⌒ヽヾゝゞ〃仝々〆〇ー―～‖¦¬ˇ˘˚∞≈≠≡√∫△◇○□＋－×÷＝≠≡≒≈∽∝＜＞≦≧≪≫√∂∇ΣΠ∏∑∫∬∭∮⌂①②③④⑤⑥⑦⑧⑨⑩⑪⑫⑬⑭⑮⑯⑰⑱⑲⑳⓪㊣℅‱‰′″℃℉ℹ⌘℗℡ℓ№℞™℠㍘㍙㍚㍛㍜㍝㍞㍟㍠㍡㍢㍣㍤㍥㍦㍧㍨㍩㍪㍫㍬㍭㍮㍯㍰〶✂✄📌📍📎📏📐✂✁✃✆☎☑☒♪♫♬♩♭♯𝟙𝟚𝟛𝟜𝟝𝟞𝟟𝟠𝟡𝟢𝟣𝟤𝟥𝟦𝟧𝟨𝟩𝟪𝟫𝗮𝗯𝗰𝗱𝗲𝗳𝗴𝗵𝗶𝗷𝗸𝗹𝗺𝗻𝗼𝗽𝗾𝗿𝘀𝘁𝘂𝘃𝘄𝘅𝘆𝘻𝘈𝘉𝘊𝘋𝘌𝘍𝘎𝘏𝘐𝘑𝘒𝘓𝘔𝘕𝘖𝘗𝘘𝘙𝘚𝘛𝘜𝘝𝘞𝘟𝘠𝘡𝙖𝙗𝙘𝙙𝙚𝙛𝙜𝙝𝙞𝙟𝙠𝙡𝙢𝙣𝙤𝙥𝙦𝙧𝙨𝙩𝙪𝙫𝙬𝙭𝙮𝙯𝘼𝘽𝘾𝘿𝙀𝙁𝙂𝙃𝙄𝙅𝙆𝙇𝙈𝙉𝙊𝙋𝙌𝙍𝙎𝙏𝙐𝙑𝙒𝙓𝙔𝙕𝗔𝗕𝗖𝗗𝗘𝗙𝗚𝗛𝗜𝗝𝗞𝗟𝗠𝗡𝗢𝗣𝗤𝗥𝗦𝗧𝗨𝗩𝗪𝗫𝗬𝗭𝔸𝔹ℂ𝔻𝔼𝔽𝔾ℍ𝕀𝕁𝕂𝕃𝕄ℕ𝕅ℕ𝕆ℙ𝕈𝕉ℝ𝕊𝕋𝕌𝕍𝕎𝕏𝕐𝕑𝔸𝔹ℂ𝔻𝔼𝔽𝔾ℍ𝕀𝕁𝕂𝕃𝕄ℕ𝕅ℕ𝕆ℙ𝕈𝕉ℝ𝕊𝕋𝕌𝕍𝕎𝕏𝕐𝕑]`)

	tn.emojiRegex = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{25AA}\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2614}\x{2615}\x{2648}-\x{2653}\x{267F}\x{2693}\x{26A1}\x{26AA}\x{26AB}\x{26BD}\x{26BE}\x{26C4}\x{26C5}\x{26CE}\x{26D4}\x{26EA}\x{26F2}\x{26F3}\x{26F5}\x{26FA}\x{26FD}\x{2702}\x{2705}\x{2708}-\x{270D}\x{270F}\x{2712}\x{2714}\x{2716}\x{271D}\x{2721}\x{2728}\x{2733}\x{2734}\x{2744}\x{2747}\x{274C}\x{274E}\x{2753}-\x{2755}\x{2757}\x{2763}\x{2764}\x{2795}\x{2796}\x{2797}\x{27A1}\x{27B0}\x{27BF}\x{2934}\x{2935}\x{2B05}-\x{2B07}\x{2B1B}\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}]`)

	tn.zeroWidthRegex = regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{FEFF}\x{2028}\x{2029}\x{00AD}\x{2060}\x{180E}]`)
}

func (tn *TextNormalizer) TraditionalToSimplified(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if simp, exists := tn.traditionalToSimplified[r]; exists {
			result = append(result, simp)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) SimplifiedToTraditional(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if trad, exists := tn.simplifiedToTraditional[r]; exists {
			result = append(result, trad)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) HalfwidthToFullwidth(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if full, exists := tn.halfwidthToFullwidth[r]; exists {
			result = append(result, full)
		} else if r >= 0x21 && r <= 0x7E {
			result = append(result, r+0xFEE0)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) FullwidthToHalfwidth(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if half, exists := tn.fullwidthToHalfwidth[r]; exists {
			result = append(result, half)
		} else if r >= 0xFF01 && r <= 0xFF5E {
			result = append(result, r-0xFEE0)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) RemoveInterleavedChars(text string) string {
	return tn.interleavedCharsRegex.ReplaceAllString(text, "")
}

func (tn *TextNormalizer) RemoveEmojis(text string) string {
	return tn.emojiRegex.ReplaceAllString(text, "")
}

// ExtractEmojis 从文本中提取所有 emoji 字符
// 在 RemoveEmojis 之前调用，用于独立检查 emoji 违规
func (tn *TextNormalizer) ExtractEmojis(text string) []rune {
	matches := tn.emojiRegex.FindAllString(text, -1)
	emojis := make([]rune, 0, len(matches))
	for _, m := range matches {
		for _, r := range m {
			emojis = append(emojis, r)
		}
	}
	return emojis
}

func (tn *TextNormalizer) RemoveZeroWidthChars(text string) string {
	return tn.zeroWidthRegex.ReplaceAllString(text, "")
}

func (tn *TextNormalizer) RemoveDuplicateChars(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	result := make([]rune, 0, len(runes))
	i := 0
	n := len(runes)

	for i < n {
		current := runes[i]
		count := 1

		for i+count < n && runes[i+count] == current {
			count++
		}

		if count >= 3 {
			result = append(result, current)
		} else {
			for j := 0; j < count; j++ {
				result = append(result, current)
			}
		}

		i += count
	}

	return string(result)
}

func (tn *TextNormalizer) RemovePunctuation(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if !unicode.IsPunct(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) RemoveSpaces(text string) string {
	return strings.ReplaceAll(text, " ", "")
}

func (tn *TextNormalizer) RemoveNumbers(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if !unicode.IsDigit(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) RemoveSpecialChars(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

func (tn *TextNormalizer) NormalizeText(text string) string {
	text = tn.FullwidthToHalfwidth(text)

	text = tn.TraditionalToSimplified(text)

	text = tn.RemoveZeroWidthChars(text)

	text = tn.RemoveEmojis(text)

	text = tn.RemoveInterleavedChars(text)

	text = tn.RemoveDuplicateChars(text)

	text = strings.Join(strings.Fields(text), " ")

	text = strings.ToLower(text)

	return text
}

func (tn *TextNormalizer) NormalizeTextWithOriginal(text string) (normalized string, original string) {
	return tn.NormalizeText(text), text
}
