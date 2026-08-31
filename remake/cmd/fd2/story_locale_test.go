package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestReviewedOpeningTranslationsUseCanonicalLineIDs(t *testing.T) {
	wants := map[string]map[string]string{
		"ja": {
			"legacy/line/cac6c4f06aba/scenes/0/lines/3/text":          "ティエノ、やっと追いついたぞ！これでどう逃げるつもりだ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/speaker-name":  "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/text":          "あなたたちは俺にどんな恨みがあって、どうして何としても殺そうとするんだ！？",
			"legacy/line/cac6c4f06aba/scenes/0/lines/5/text":          "俺たちの隊長が来るまで生きていられたら、本人に聞くんだな！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/speaker-name":  "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/text":          "くそっ！お前たちと戦ってやる！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/speaker-name":  "シリア",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/text":          "あら！見て、橋の上で誰かが喧嘩を始めようとしてるわ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/speaker-name":  "ソール",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/text":          "大勢で一人を相手にするなんて不公平だ！行こう、アレス。あの一人で戦っている奴を助けるぞ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/9/speaker-name":  "アレス",
			"legacy/line/cac6c4f06aba/scenes/0/lines/10/speaker-name": "ソール",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/speaker-name": "アレス",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/text":         "行け、行け！お前一人で戦うのを黙って見ているわけにはいかない。お前と知り合ったのが運の尽きだ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/speaker-name": "シリア",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/text":         "私も行く！私も行くわ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/speaker-name": "ハノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/text":         "それなら、みんなで行こう！さあ、行くぞ！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/speaker-name": "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/text":         "もしこの若者たちが助けてくれなかったら、私はとっくに死んでいた！だが、まだ生きているうちに一つ聞きたい。いったい誰がお前に私を殺せと命じた？",
			"legacy/line/cac6c4f06aba/scenes/0/lines/17/speaker-name": "シリア",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/speaker-name": "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/text":         "お、お前、どうしてこのことを知っているんだ？",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/speaker-name": "シリア",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/text":         "それは秘密よ！でも安心して。私たちがみんなで助けてあげるわ。",
			"legacy/line/15b3c967fb2b/scenes/0/lines/0":               "アレス、もう一度勝負しよう！昨日負けたのが悔しくてたまらない！",
			"legacy/line/15b3c967fb2b/scenes/2/lines/7":               "ユニさん、まずは僕と一緒に戻ろう。数日後にはマラ大陸へ出発するよ。心配しないで、必ず無事に家へ送り届けるから。",
			"legacy/line/ae86adb52dac/scenes/0/lines/0":               "父上、ソールです。謁見に参りました。",
			"legacy/line/ae86adb52dac/scenes/1/lines/17":              "それじゃ仕方ないな。もう引き受けるしかないのか？",
			"legacy/line/7ecb566a60db/scenes/1/lines/5":               "ただの強盗じゃない。こいつらはマラ大陸の沿岸を荒らし、人殺しも略奪も何でもする海賊だって聞いている。",
			"legacy/line/7ecb566a60db/scenes/7/lines/8":               "ぼ、僕はハノです。これから、よろしくお願いします。",
			"legacy/line/7ecb566a60db/scenes/1/lines/13":              "ならば容赦するな！　行け！",
			"legacy/line/7ecb566a60db/scenes/2/lines/10":              "若者たちは元気だな！　お前たち、行くぞ！",
			"legacy/line/7ecb566a60db/scenes/3/lines/12":              "くそっ！　あの小僧二人、俺を完全に無視しやがって！　殺せ！　一人も逃がすな！",
			"legacy/line/7ecb566a60db/scenes/4/lines/4":               "ソール、これから人の縄張りに入るんだから、せめて礼儀正しくしなよ！",
			"legacy/line/7ecb566a60db/scenes/7/lines/12":              "お父さん、安心してください！　それじゃ行きます！",
		},
		"en": {
			"legacy/line/cac6c4f06aba/scenes/0/lines/3/text":          "Tino, I've finally caught up with you! Let's see how you escape this time!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/speaker-name":  "Tino",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/text":          "What grudge do you people hold against me? Why are you so determined to kill me?!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/5/text":          "If you live until our captain gets here, ask him yourself!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/speaker-name":  "Tino",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/text":          "Damn it! I'll fight you!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/speaker-name":  "Celia",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/text":          "Oh! Look, someone's about to start a fight on the bridge!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/speaker-name":  "Sol",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/text":          "It's not fair to gang up on one person! Come on, Ares—let's help that lone fighter!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/9/speaker-name":  "Ares",
			"legacy/line/cac6c4f06aba/scenes/0/lines/10/speaker-name": "Sol",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/speaker-name": "Ares",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/text":         "Go, go! I can't just stand by and watch you fight alone. Knowing you has been nothing but trouble!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/speaker-name": "Celia",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/text":         "I'm going too! Me too!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/speaker-name": "Hano",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/text":         "In that case, everyone's going. Let's go!",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/speaker-name": "Tino",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/text":         "If these young men hadn't helped me, I'd already be dead! But since I'm still alive, I have one question: who ordered you to kill me?",
			"legacy/line/cac6c4f06aba/scenes/0/lines/17/speaker-name": "Celia",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/speaker-name": "Tino",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/text":         "Y-you know about this?",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/speaker-name": "Celia",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/text":         "That’s a secret! But don’t worry—we’ll all help you.",
			"legacy/line/15b3c967fb2b/scenes/0/lines/0":               "Ares, let's have another match! Losing to you yesterday still bothers me!",
			"legacy/line/ae86adb52dac/scenes/0/lines/0":               "I am Sol. I have come to pay my respects, Father.",
			"legacy/line/ae86adb52dac/scenes/1/lines/16":              "Ever since I was a child, I wanted to go abroad and have grand adventures—not sit on a throne for the rest of my life. I am only Father's adopted son and have no royal blood, so I always thought Dean would inherit the throne and I would be free. But now...",
			"legacy/line/7ecb566a60db/scenes/3/lines/4":               "That must be the pirate leader! At last, a worthy opponent has appeared. Leave this one to me!",
			"legacy/line/7ecb566a60db/scenes/7/lines/8":               "My name is... Hano. I hope we can get along.",
			"legacy/line/7ecb566a60db/scenes/1/lines/13":              "Then show them no mercy! Attack!",
			"legacy/line/7ecb566a60db/scenes/3/lines/10":              "I can't stand this guy. Sol, let's stop arguing and take him down together!",
			"legacy/line/7ecb566a60db/scenes/3/lines/12":              "Damn it! Those two brats don't respect me at all! Kill them! Don't let a single one escape!",
			"legacy/line/7ecb566a60db/scenes/7/lines/5":               "Of course, sir! There’s no problem with that at all! Isn’t that right, Yuni?",
			"legacy/line/7ecb566a60db/scenes/8/lines/1":               "Dad! Dad!",
		},
		"zh-Hans": {
			"legacy/line/cac6c4f06aba/scenes/0/lines/3/text":          "铁诺，终于追上你了吧！这下看你怎么逃！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/speaker-name":  "铁诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/4/text":          "你们到底跟我有什么仇，非要杀了我不可？！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/5/text":          "要是你能活到我们队长赶到，再自己问他吧！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/speaker-name":  "铁诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/6/text":          "可恶！我和你们拼了！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/speaker-name":  "希莉亚",
			"legacy/line/cac6c4f06aba/scenes/0/lines/7/text":          "哎呀！看，桥上有人准备要打架了！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/speaker-name":  "索尔",
			"legacy/line/cac6c4f06aba/scenes/0/lines/8/text":          "以多打少，这太不公平了！走，亚雷斯，我们去帮那个落单的人！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/9/speaker-name":  "亚雷斯",
			"legacy/line/cac6c4f06aba/scenes/0/lines/10/speaker-name": "索尔",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/speaker-name": "亚雷斯",
			"legacy/line/cac6c4f06aba/scenes/0/lines/11/text":         "去，去！我总不能看着你一个人去拼命，认识你算我倒霉！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/speaker-name": "希莉亚",
			"legacy/line/cac6c4f06aba/scenes/0/lines/12/text":         "我也去！我也去！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/speaker-name": "哈诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/13/text":         "既然如此，那么大家都去吧！走啰！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/speaker-name": "铁诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/15/text":         "如果不是这些年轻人帮忙的话，我早就没命了！不过既然我还活着，我还是要问你一个问题：到底是谁命令你来杀我？",
			"legacy/line/cac6c4f06aba/scenes/0/lines/17/speaker-name": "希莉亚",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/speaker-name": "铁诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/18/text":         "你……你怎么会知道这件事？",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/speaker-name": "希莉亚",
			"legacy/line/cac6c4f06aba/scenes/0/lines/19/text":         "秘密！不过你可以放心，我们都会帮你的。",
			"legacy/line/15b3c967fb2b/scenes/1/lines/13":              "悠妮?好名字。悠妮小姐,你怎么会记不得怎么来到这里的?",
			"legacy/line/7ecb566a60db/scenes/0/lines/2":               "太好了。悠妮，你……嗯，坐了这么久的船，有点累吧？",
			"legacy/line/7ecb566a60db/scenes/4/lines/2":               "我们是亚克斯王国的海岸巡防队，消灭肆虐沿海的海盗本是我们的职责，这些海盗就交给我们来处理，请各位放心！",
			"legacy/line/7ecb566a60db/scenes/1/lines/13":              "那就杀无赦！上啊！",
			"legacy/line/7ecb566a60db/scenes/2/lines/2":               "什么？待我看看……啊哈，这不是海盗在打劫旅客吗？居然敢在我们门前抢人，胆子不小啊！",
			"legacy/line/7ecb566a60db/scenes/3/lines/12":              "可恶啊！这两个小子全不把我放在眼里！给我杀！一个都别放过！",
			"legacy/line/7ecb566a60db/scenes/7/lines/1":               "哪里！我和这小子住在岛上，除了偶尔外出游历，平时也没什么事。帮你们打几个海盗，不算什么啦！对了，说到这个，我这老头子倒有个请求。",
		},
	}
	for localeID, entries := range wants {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		for lineID, want := range entries {
			got, err := content.StoryText(lineID)
			if err != nil || got != want {
				t.Fatalf("%s %s=%q err=%v, want %q", localeID, lineID, got, err, want)
			}
		}
		assertReviewedContentEntries(t, localeID, entries)
	}
}

func TestReviewedChapterThreeSceneOneLinesOneToFive(t *testing.T) {
	wants := map[string]map[string]string{
		"en": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/1/speaker-name": "Sol",
			"legacy/line/cac6c4f06aba/scenes/1/lines/1/text":         "What a shame. We did everything we could, but we still couldn’t save him…",
			"legacy/line/cac6c4f06aba/scenes/1/lines/2/speaker-name": "Celia",
			"legacy/line/cac6c4f06aba/scenes/1/lines/2/text":         "Perhaps it’s for the best. He can go underground and be reunited with his beloved, without having to keep fleeing those who are hunting him…",
			"legacy/line/cac6c4f06aba/scenes/1/lines/3/text":         "Celia, what is your true identity?",
			"legacy/line/cac6c4f06aba/scenes/1/lines/4/speaker-name": "Celia",
			"legacy/line/cac6c4f06aba/scenes/1/lines/4/text":         "That’s a secret! Anyway, let’s leave as soon as we’ve buried him. Otherwise, there may be more trouble ahead…",
			"legacy/line/cac6c4f06aba/scenes/1/lines/5/speaker-name": "Sol",
			"legacy/line/cac6c4f06aba/scenes/1/lines/5/text":         "Let’s do that.",
		},
		"ja": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/1/text":         "残念だ。俺たちも力を尽くしたが、結局彼を助けられなかった……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/2/speaker-name": "シリア",
			"legacy/line/cac6c4f06aba/scenes/1/lines/2/text":         "これでよかったのよ。地下で愛する人と再会できるし、もう追っ手から逃げ回らなくてもいい……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/4/text":         "それは秘密よ！とにかく、彼を埋葬したらすぐにここを離れましょう。そうしないと、まだ面倒なことが待っているかもしれないわ……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/5/text":         "そうしよう。",
		},
		"zh-Hans": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/1/text": "真遗憾，我们已经尽力了，结果还是帮不了他……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/2/text": "也好，让他到地下和爱人相聚，也不用再四处躲避别人的追杀……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/3/text": "希莉亚，你的真实身份究竟是什么？",
			"legacy/line/cac6c4f06aba/scenes/1/lines/4/text": "秘密！总之，我们安葬他之后就赶快离开这里吧，否则麻烦可能还在后头呢！",
			"legacy/line/cac6c4f06aba/scenes/1/lines/5/text": "就这么办吧！",
		},
	}
	for localeID, entries := range wants {
		if _, err := loadOfficialLocaleContent(localeID); err != nil {
			t.Fatal(err)
		}
		assertReviewedContentEntryIDs(t, localeID, entries)
	}
}

func TestReviewedChapterThreeSceneOneLinesSixToTen(t *testing.T) {
	wants := map[string]map[string]string{
		"en": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/6/text":         "Thank you all for your help. Without you, I couldn’t have escaped this pursuit. But Grey won’t give up. I still can’t find anywhere on this land where I can live in peace…",
			"legacy/line/cac6c4f06aba/scenes/1/lines/8/speaker-name": "Tino",
			"legacy/line/cac6c4f06aba/scenes/1/lines/9/text":         "That’s a secret! Anyway, you’re joining us.",
			"legacy/line/cac6c4f06aba/scenes/1/lines/10/text":        "Hmph! What a mischievous, mysterious girl.",
		},
		"ja": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/6/text":         "皆さん、助けてくれてありがとう。おかげで追っ手から逃げられた。だが、グレイはこれで諦めるはずがない。この土地では、まだ安心して暮らせる場所が見つからない……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/8/speaker-name": "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/1/lines/9/text":         "それは秘密よ！とにかく、あなたは私たちの仲間になるの。",
			"legacy/line/cac6c4f06aba/scenes/1/lines/10/text":        "ふん！いかにも謎めいていて、まったく食えない娘だな。",
		},
		"zh-Hans": {
			"legacy/line/cac6c4f06aba/scenes/1/lines/6/text":         "多谢各位的帮助，我才能逃过这次追杀。不过葛雷是不会就此罢手的，我在这片土地上还是找不到安居之所……",
			"legacy/line/cac6c4f06aba/scenes/1/lines/8/speaker-name": "铁诺",
			"legacy/line/cac6c4f06aba/scenes/1/lines/9/text":         "秘密！反正你加入我们就是了。",
			"legacy/line/cac6c4f06aba/scenes/1/lines/10/text":        "哼！一副故作神秘的样子，真是个古灵精怪的丫头。",
		},
	}
	for localeID, entries := range wants {
		if _, err := loadOfficialLocaleContent(localeID); err != nil {
			t.Fatal(err)
		}
		assertReviewedContentEntryIDs(t, localeID, entries)
	}
}

func TestReviewedChapterTwoCampaignTranslationsAndItemEntities(t *testing.T) {
	wants := map[string]map[string]string{
		"ja": {
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.preparation_ch02.prompt":       "戦場に入りますか？",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.retreat_ch02.lines.0.text":     "撤退！まず態勢を立て直し、反撃の機会を探そう……",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.rumor_ch02.lines.0.text":       "ここには店頭に並べない良い物があるらしい……（酒場の前でShift+F1キー）",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.0.name": "ブロードソード",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.1.name": "ハルバード",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.2.name": "ウォーハンマー",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.0.label":     "酒場：情報を聞く",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.2.label":     "出口：出撃準備",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.town":                "ロード",
			"legacy/story/5bf145f103fe/ch02/title":                                                  "第2章 ― ロードの盗賊団",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/speaker-name":                                "アレス",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/text":                                        "ロードに到着した。沿岸で最も栄えた町らしい。ここで少し休み、酒場で情報を集めよう……",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/text":                                        "アレス、君の情報は間違っていたようだ。ここには人っ子一人いない。まるでゴーストタウンだ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/speaker-name":                                "ユニ",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/text":                                        "こ、怖い……",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/speaker-name":                                "ハノ",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/text":                                        "おかしいな。前に親父と酒を買いに来たときは、町はずいぶん賑わっていたのに……",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/speaker-name":                                "村人",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/text":                                        "盗賊だ！早く逃げろ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/text":                                        "待ってくれ！俺たちは盗賊じゃない！",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/speaker-name":                                "村人",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/text":                                        "盗賊ではないのですか？では、何をしにここへ？",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/speaker-name":                                "アレス",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/text":                                        "我々は通りすがりで、町で少し休もうと思っただけだ！事情を詳しく話してくれ。力になれるかもしれない！",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/speaker-name":                                "村人",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/text":                                        "はあ、君たちのような若造に何ができる？凶悪な盗賊団がこの町を襲うと予告し、町の人々は皆逃げた。私たちも財物を少し持ち出したら、すぐここを離れるつもりだ。君たちも早く逃げなさい。盗賊団に出会ったら大変だ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/speaker-name":                                "ガイア",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/text":                                        "『……！！！』",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/speaker-name":                               "盗賊B",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/text":                                       "ヤッホー！予告どおり時間ぴったりに来たぜ！おや？まだ命知らずが残っているのか？",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/speaker-name":                               "村人",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/text":                                       "ああ、盗賊だ！もうおしまいだ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/speaker-name":                               "盗賊B",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/text":                                       "俺たちが来たら町に残るなと言っただろう？死にたいのか！",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/speaker-name":                               "村人",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/text":                                       "ああ、盗賊様、どうか見逃してください。もう二度としません！",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/speaker-name":                               "アレス",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/text":                                       "なんて乱暴な盗賊たちだ！ソール、奴らに思い知らせてやろう！",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/text":                                       "何が『教訓』だ！こいつらに悔い改める暇など与えず、皆殺しにしてやる！",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/speaker-name":                               "ハノ",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/text":                                       "賛成だ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/speaker-name":                               "盗賊B",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/text":                                       "おいおい、あの若造ども、俺たちを皆殺しにするつもりらしいぞ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/speaker-name":                               "盗賊C",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/text":                                       "こ、怖い！俺は怖いよ！",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/speaker-name":                               "盗賊B",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/text":                                       "そういうことなら、少し遊んでやるか！行くぞ！",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/speaker-name":                                "盗賊L",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/text":                                        "おい、兄弟たちは何をしている？もう片付けたはずだろう！",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/speaker-name":                                "盗賊M",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/text":                                        "どうやら、別の連中と激しくやり合っているようだな！",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/speaker-name":                                "盗賊N",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/text":                                        "逃げてくる残党がこちらへ向かっているぞ！先に奴らを片付けよう！",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/speaker-name":                                "村人",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/text":                                        "助けて！ここにも盗賊が現れた！",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/speaker-name":                                "アレス",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/text":                                        "くそっ、敵が二手に分かれていたとは……！",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/text":                                        "とにかく、罪のない住民たちを救う方法を考えなければ！",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/speaker-name":                                "アレス",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/text":                                        "どうしよう、ソール……！",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/text":                                        "おい、君たち！南東へ逃げろ！",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/speaker-name":                                "村人",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/text":                                        "わかった！みんな、南東へ逃げよう！",
			"legacy/line/5bf145f103fe/scenes/1/lines/9/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/speaker-name":                               "村人",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/text":                                       "ご協力ありがとうございました。これはほんの気持ちです。どうぞお受け取りください。",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/speaker-name":                               "村人",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/text":                                       "助けていただき、危機を逃れることができました。ありがとうございました。",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/speaker-name":                                "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/text":                                        "すごい！すごい！本当に見事な戦いだった！",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/text":                                        "えっ？このおてんば娘、どこから出てきたんだ？",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/speaker-name":                                "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/text":                                        "失礼ね、私はおてんば娘なんかじゃないわ。さっきあなたたちが戦っている間、ずっと家の中から見ていたの。本当に見応えのある戦いだったわ。",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/speaker-name":                                "アレス",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/text":                                        "お嬢さん、ずいぶん度胸があるな。俺たちに何かできることはあるか？",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/speaker-name":                                "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/text":                                        "大したことじゃないわ。ただ、あなたたちほどの腕前なら、盗賊退治だけが目的ではないのでしょう？",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/text":                                        "そういうことだ。この少女の記憶喪失を治し、家へ送り届けたい。",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/speaker-name":                                "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/text":                                        "これはどういうこと？",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/text":                                        "簡単に言うと、森でこの少女ユニを見つけたんだ……",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/speaker-name":                                "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/text":                                        "「見つけた」？",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/speaker-name":                                "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/text":                                        "いや、正確にはユニを見つけたんだ。彼女は記憶を失っているが、マラ大陸の人、ひょっとするとどこかの王国の王女かもしれない。だから海を渡って彼女を故郷へ送り届けようとしている。",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/speaker-name":                               "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/text":                                       "おかしいわ。この大陸にはアクス王国しかなくて、その王国の王女は……",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/text":                                       "それで？",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/speaker-name":                               "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/text":                                       "な、何でもないわ。そうだ、記憶喪失を治せる人がいる場所を知っているから、あなたたちを案内してあげる。ただし……条件が一つあるの。",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/text":                                       "まず聞かせてくれ。できる限り何とかしよう。",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/speaker-name":                               "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/text":                                       "実は簡単なことよ。私も一緒に連れて行ってほしいの。",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/speaker-name":                               "アレス",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/text":                                       "それは……",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/text":                                       "お嬢さん、これは遊びじゃない。若い女の子が俺たちについて来たら、とんでもない目に遭うぞ。",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/speaker-name":                               "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/text":                                       "私の腕を甘く見ないで。大変な目に遭うのは、あなたのほうよ。どう？ 私の条件を受け入れる？",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/speaker-name":                               "ハノ",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/text":                                       "ソール兄貴、盗賊がいつ戻って来るか分からない以上、女の子をここに置いていくのは危険です。ひとまず彼女を連れて行ったほうがいいと思います。",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/speaker-name":                               "アレス",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/text":                                       "そうだな、ソール。ユニのことは一刻も早く解決しなければならない。一人増えたくらい、俺たちが少し余計に気を配ればいい。",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/text":                                       "分かった。ただし、お嬢さん、よく聞け。案内役は君だ。俺たちの足を引っ張るな。分かったな？",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/speaker-name":                               "シリア",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/text":                                       "問題ないよ！ さあ行こう、私が案内するから。こっち！",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/speaker-name":                               "ソール",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/text":                                       "まったく、仕方ないな……。",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/speaker-name":                                "シリア",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/text":                                        "この橋を渡れば、医術で名高いセラ村まであと少しです。あの村の司祭の治療魔法はとても神秘的で、ユニの記憶喪失も治せるはずです。",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/speaker-name":                                "ソール",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/text":                                        "そうだといいな……えっ……",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/speaker-name":                                "ティエノ",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/text":                                        "……ふう、今回はもう追いつけそうにない……",
		},
		"en": {
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.preparation_ch02.prompt":       "Enter the battlefield?",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.retreat_ch02.lines.0.text":     "Fall back! Let's regroup and look for a chance to counterattack...",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.rumor_ch02.lines.0.text":       "I hear there are good things here that aren't put on display... (Press Shift+F1 in front of the tavern.)",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.0.name": "Broadsword",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.1.name": "Halberd",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.2.name": "War Hammer",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.0.label":     "Tavern: Ask Around",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.2.label":     "Exit: Battle Preparations",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.town":                "Rhodes",
			"legacy/story/5bf145f103fe/ch02/title":                                                  "Chapter II — The Bandits of Rhodes",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/speaker-name":                                "Ares",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/text":                                        "We have arrived in Rhodes. I hear this is the most prosperous town on the coast. We can rest here and ask around at the tavern for news...",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/text":                                        "Ares, I think your information was wrong. There isn't a soul here—it looks like a ghost town!",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/speaker-name":                                "Yuni",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/text":                                        "This is scary...",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/speaker-name":                                "Hano",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/text":                                        "That's strange. The last time Dad and I came here to buy drinks, the town was bustling. How could this have happened...?",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/speaker-name":                                "Villager",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/text":                                        "The bandits are here! Run!",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/text":                                        "Wait! We're not bandits!",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/speaker-name":                                "Villager",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/text":                                        "You're not bandits? Then what are you doing here?",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/speaker-name":                                "Ares",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/text":                                        "We're only passing through and hoped to rest in town! Tell us what happened; perhaps we can help!",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/speaker-name":                                "Villager",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/text":                                        "Alas, what can you youngsters do? The vicious bandits announced they would raid this town, so everyone fled. We took a few valuables and are about to leave as well. You should flee too—running into those bandits would be disastrous!",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/speaker-name":                                "Gaia",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/text":                                        "…!!!",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/speaker-name":                               "Bandit B",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/text":                                       "Yahoo! We've arrived right on schedule, just as announced! Huh? Some fools are still here?",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/speaker-name":                               "Villager",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/text":                                       "The bandits are here! We're doomed!",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/speaker-name":                               "Bandit B",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/text":                                       "Didn't we say no one was to remain in town when we arrived? Are you trying to get yourselves killed?",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/speaker-name":                               "Villager",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/text":                                       "Ah, Lord Bandit, please spare us. We won't do it again!",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/speaker-name":                               "Ares",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/text":                                       "These bandits are outrageous! Sol, let's teach them a lesson!",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/text":                                       "What lesson? We'll give these scum no chance to repent—kill them all!",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/speaker-name":                               "Hano",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/text":                                       "I agree!",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/speaker-name":                               "Bandit B",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/text":                                       "Oh dear! Those youngsters say they're going to kill us all!",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/speaker-name":                               "Bandit C",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/text":                                       "This is terrifying! I'm scared!",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/speaker-name":                               "Bandit B",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/text":                                       "In that case, let's have a little fun with them! Attack!",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/speaker-name":                                "Bandit L",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/text":                                        "Hey, brothers, what are you doing? You should have finished by now!",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/speaker-name":                                "Bandit M",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/text":                                        "Looks like they're having quite a fight with someone else!",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/speaker-name":                                "Bandit N",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/text":                                        "I saw some stragglers fleeing this way! Let's deal with them first!",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/speaker-name":                                "Villager",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/text":                                        "Help! More bandits have appeared here!",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/speaker-name":                                "Ares",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/text":                                        "Damn, I didn't expect the enemy to split into two groups...!",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/text":                                        "Either way, we have to find a way to save those innocent townspeople!",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/speaker-name":                                "Ares",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/text":                                        "What should we do, Sol...?!",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/text":                                        "Hey, you! Run southeast!",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/speaker-name":                                "Villagers",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/text":                                        "All right! Everyone, run southeast!",
			"legacy/line/5bf145f103fe/scenes/1/lines/9/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/speaker-name":                               "Villagers",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/text":                                       "Thank you for your help. Please accept this small token of our gratitude.",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/speaker-name":                               "Villagers",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/text":                                       "Thank you for your help. You saved us from disaster.",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/speaker-name":                                "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/text":                                        "Bravo! Bravo! That was an incredible fight!",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/text":                                        "Huh? Where did this wild girl come from?",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/speaker-name":                                "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/text":                                        "Watch your manners! I'm not some wild girl. I watched from inside the house while you fought—it was quite a show.",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/speaker-name":                                "Ares",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/text":                                        "Miss, you're quite brave. Is there anything we can do for you?",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/speaker-name":                                "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/text":                                        "It's nothing. I was just wondering—fighters as skilled as you must have come here for more than fighting bandits, right?",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/text":                                        "That's right. We want to cure this girl's amnesia and take her home.",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/speaker-name":                                "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/text":                                        "What's going on?",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/text":                                        "In short, we found this girl, Yuni, in the forest...",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/speaker-name":                                "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/text":                                        "Found?",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/speaker-name":                                "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/text":                                        "No, no—we found Yuni. She has lost her memory, but we believe she's from Mara Continent, perhaps even a princess of some kingdom. That's why we crossed the sea to bring her home.",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/speaker-name":                               "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/text":                                       "That's strange. There is only one kingdom on this continent—the Kingdom of Aks—and its princess is...",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/text":                                       "And?",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/speaker-name":                               "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/text":                                       "N-Nothing. I know someone who can cure amnesia. I can take you there, but... I have one condition.",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/text":                                       "Let's hear it first. We'll try.",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/speaker-name":                               "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/text":                                       "It's simple, really. You just have to take me with you.",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/speaker-name":                               "Ares",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/text":                                       "That...",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/text":                                       "Miss, this isn't a game. You're a young girl—if you follow us, you could end up in serious trouble.",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/speaker-name":                               "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/text":                                       "Don't underestimate what I can do, or you'll be the one in trouble. Well? Do you accept my condition?",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/speaker-name":                               "Hano",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/text":                                       "Brother Sol, we don't know when the bandits might return. It's too dangerous to leave a girl here alone, so I think we'd better take her with us for now.",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/speaker-name":                               "Ares",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/text":                                       "That's right, Sol. We need to resolve Yuni's situation as soon as possible. Having one more girl along just means we'll have to keep an extra eye on her.",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/text":                                       "All right. But listen carefully, miss: you're responsible for leading the way, so don't cause us any trouble. Got it?",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/speaker-name":                               "Celia",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/text":                                       "No problem! Come on, let's go. I'll lead the way—this way!",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/speaker-name":                               "Sol",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/text":                                       "Honestly, what a handful...",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/speaker-name":                                "Celia",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/text":                                        "After we cross this bridge, we'll soon reach Sela Village, famous for its healing arts. The priests' healing magic there is truly remarkable; it should be able to cure Yuni's amnesia.",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/speaker-name":                                "Sol",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/text":                                        "I hope so... Huh...",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/speaker-name":                                "Tino",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/text":                                        "...Whew, I don't think I'll catch up this time...",
		},
		"zh-Hans": {
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.preparation_ch02.prompt":       "要进入战场吗？",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.retreat_ch02.lines.0.text":     "撤退！先回头整顿，再找机会反攻……",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.rumor_ch02.lines.0.text":       "听说这里有不摆在台面上的好东西……（酒馆前按 Shift+F1 键）",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.0.name": "阔剑",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.1.name": "长戟",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.shop_ch02_secret.goods.2.name": "钉头锤",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.0.label":     "酒馆：打听消息",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.options.2.label":     "出口：出战整备",
			"legacy.json.remake.assets.scenarios.campaign_full.nodes.town_ch02.town":                "罗德镇",
			"legacy/story/5bf145f103fe/ch02/title":                                                  "第二章——罗德镇的强盗团",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/speaker-name":                                "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/0/lines/0/text":                                        "我们已经抵达罗德镇了。听说这里是沿岸最繁荣的城镇，我们可以在这里歇一下，顺便到酒馆里打听一下消息……",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/0/lines/1/text":                                        "亚雷斯，我看你的情报有误，这里连半个人都没有，倒像是一座鬼城！",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/speaker-name":                                "悠妮",
			"legacy/line/5bf145f103fe/scenes/0/lines/2/text":                                        "好可怕……",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/speaker-name":                                "哈诺",
			"legacy/line/5bf145f103fe/scenes/0/lines/3/text":                                        "这可怪了，上次我和老爹来这里买酒的时候，镇上还热闹得很啊！怎么会……",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/speaker-name":                                "村民",
			"legacy/line/5bf145f103fe/scenes/0/lines/4/text":                                        "啊！强盗来了！快逃啊！",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/0/lines/5/text":                                        "等等！我们不是强盗！",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/speaker-name":                                "村民",
			"legacy/line/5bf145f103fe/scenes/0/lines/6/text":                                        "你们不是强盗，那你们是来干什么的？",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/speaker-name":                                "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/0/lines/7/text":                                        "我们不过是路过此地，想在镇上休息一下而已！请把事情说清楚，说不定我们可以助各位一臂之力！",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/speaker-name":                                "村民",
			"legacy/line/5bf145f103fe/scenes/0/lines/8/text":                                        "唉！你们这些小毛头能做什么？穷凶极恶的强盗团预告要在此时洗劫本镇，镇上的人都逃光了。我们是趁机带走一些财物的，马上也要逃离此地。你们也快逃吧，要是碰上那群强盗就糟啦！",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/speaker-name":                                "盖亚",
			"legacy/line/5bf145f103fe/scenes/0/lines/9/text":                                        "『……！！！』",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/speaker-name":                               "强盗B",
			"legacy/line/5bf145f103fe/scenes/0/lines/10/text":                                       "呀呼！我们照预告准时抵达！咦？居然还有不怕死的没走？",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/speaker-name":                               "村民",
			"legacy/line/5bf145f103fe/scenes/0/lines/11/text":                                       "啊！强盗来了！完啦！",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/speaker-name":                               "强盗B",
			"legacy/line/5bf145f103fe/scenes/0/lines/12/text":                                       "我们不是说过，我们来的时候不准有人留在镇上吗？你们是活得不耐烦了！",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/speaker-name":                               "村民",
			"legacy/line/5bf145f103fe/scenes/0/lines/13/text":                                       "……啊，强盗大人，请放我们一马，以后不敢了！",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/speaker-name":                               "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/0/lines/14/text":                                       "这些强盗真猖狂啊！索尔，我们来教训他们一顿！",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/0/lines/15/text":                                       "什么教训！不给这些家伙悔改的机会，把他们全宰了再说！",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/speaker-name":                               "哈诺",
			"legacy/line/5bf145f103fe/scenes/0/lines/16/text":                                       "我赞成！",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/speaker-name":                               "强盗B",
			"legacy/line/5bf145f103fe/scenes/0/lines/17/text":                                       "哎呀！那群小伙子说要把我们全宰了呢！",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/speaker-name":                               "强盗C",
			"legacy/line/5bf145f103fe/scenes/0/lines/18/text":                                       "好可怕呀！我好怕！",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/speaker-name":                               "强盗B",
			"legacy/line/5bf145f103fe/scenes/0/lines/19/text":                                       "既然如此，我们就陪他们玩个两手！上！",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/speaker-name":                                "强盗L",
			"legacy/line/5bf145f103fe/scenes/1/lines/0/text":                                        "咦！弟兄们在干什么？他们早该把事情办好了吧！",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/speaker-name":                                "强盗M",
			"legacy/line/5bf145f103fe/scenes/1/lines/1/text":                                        "好像和别人打得正起劲呢！",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/speaker-name":                                "强盗N",
			"legacy/line/5bf145f103fe/scenes/1/lines/2/text":                                        "我看到有漏网之鱼向这里逃来呢！我们先打发了他们再说！",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/speaker-name":                                "村民",
			"legacy/line/5bf145f103fe/scenes/1/lines/3/text":                                        "救命啊！这里又有强盗出现了！",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/speaker-name":                                "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/1/lines/4/text":                                        "可恶，没想到敌人居然兵分两路……！",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/1/lines/5/text":                                        "不管怎样，得想办法抢救那些无辜的居民！",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/speaker-name":                                "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/1/lines/6/text":                                        "怎么办呢？索尔……！",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/1/lines/7/text":                                        "喂！你们，往东南边逃吧！",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/speaker-name":                                "村民",
			"legacy/line/5bf145f103fe/scenes/1/lines/8/text":                                        "好的！大伙儿往东南边逃吧！",
			"legacy/line/5bf145f103fe/scenes/1/lines/9/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/speaker-name":                               "村民",
			"legacy/line/5bf145f103fe/scenes/1/lines/10/text":                                       "谢谢各位的帮助,这是我们的一点小意思,请收下吧。",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/speaker-name":                               "村民",
			"legacy/line/5bf145f103fe/scenes/1/lines/11/text":                                       "谢谢各位的帮助,让我们逃过一劫。",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/speaker-name":                                "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/0/text":                                        "好棒！好棒！打得真漂亮！",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/1/text":                                        "咦？哪里蹦出来的野丫头？",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/speaker-name":                                "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/2/text":                                        "说话客气点，我可不是野丫头。刚才你们战斗时，我一直躲在屋子里看，真是看了一场好戏。",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/speaker-name":                                "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/2/lines/3/text":                                        "小姐，你的胆量可真大。有什么我们能效劳的吗？",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/speaker-name":                                "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/4/text":                                        "也没什么啦，只是想问问你们，你们这么好的身手，应该不只是来这里打强盗的吧？",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/5/text":                                        "是这样的，我们想医治这个女孩子的失忆症，并且送她回家。",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/speaker-name":                                "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/6/text":                                        "这是怎么一回事？",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/7/text":                                        "简单地说，我们在森林捡到了悠妮这女孩……",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/speaker-name":                                "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/8/text":                                        "捡？",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/speaker-name":                                "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/9/text":                                        "不对不对，是找到了悠妮。虽然失去了记忆，但是我们相信她是这马拉大陆的人，甚至是某王国的公主也说不定，因此设法渡海送她回来。",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/speaker-name":                               "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/10/text":                                       "奇怪了，这大陆上就只有一个亚克斯王国，而这个王国的公主就……",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/11/text":                                       "就怎样？",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/speaker-name":                               "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/12/text":                                       "没……没什么啦。这样吧，我知道哪里有人能医治失忆症，可以带你们去，不过……有一个条件哦！",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/13/text":                                       "先说来听听，我们会尽量想办法的。",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/speaker-name":                               "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/14/text":                                       "其实很简单，就是要带我一起去。",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/speaker-name":                               "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/2/lines/15/text":                                       "这……",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/16/text":                                       "小姐，这可不是闹着玩的。你一个年轻姑娘跟着我们，恐怕会倒大霉哦。",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/speaker-name":                               "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/17/text":                                       "小看我的本领，你才要倒大霉呢。怎么样？接受我的条件吗？",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/speaker-name":                               "哈诺",
			"legacy/line/5bf145f103fe/scenes/2/lines/18/text":                                       "索尔大哥，现在也不知道强盗什么时候会再来，把一个女孩子就这样丢在这里太危险了，我看还是先带她离开的好。",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/speaker-name":                               "亚雷斯",
			"legacy/line/5bf145f103fe/scenes/2/lines/19/text":                                       "是啊，索尔，悠妮的事必须尽快解决。多带一个人，顶多就是我们多操点心。",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/20/text":                                       "好吧！不过小姐，你给我听好，你是负责带路的，别给我们添麻烦。知道吗？",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/speaker-name":                               "希莉亚",
			"legacy/line/5bf145f103fe/scenes/2/lines/21/text":                                       "没问题！走吧走吧，我来带路！往这边！",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/speaker-name":                               "索尔",
			"legacy/line/5bf145f103fe/scenes/2/lines/22/text":                                       "真是没办法，唉！",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/speaker-name":                                "希莉亚",
			"legacy/line/cac6c4f06aba/scenes/0/lines/0/text":                                        "过了这座桥，就快到以医术闻名的赛拉村了。这个村子的祭司医疗法术非常神奇，应该能治好悠妮的失忆症。",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/speaker-name":                                "索尔",
			"legacy/line/cac6c4f06aba/scenes/0/lines/1/text":                                        "希望如此……咦……",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/speaker-name":                                "铁诺",
			"legacy/line/cac6c4f06aba/scenes/0/lines/2/text":                                        "……呼，这次大概是追不上了……",
		},
	}
	itemWants := map[string]map[int]string{
		"ja":      {1: "ブロードソード", 22: "ハルバード", 53: "ウォーハンマー", 128: "布の服", 129: "旅装", 132: "レザーアーマー", 165: "魔術師のローブ", 192: "薬草", 193: "回復薬"},
		"en":      {1: "Broadsword", 22: "Halberd", 53: "War Hammer", 128: "Cloth Garb", 129: "Travel Garb", 132: "Leather Armor", 165: "Mage Robe", 192: "Herb", 193: "Recovery Potion"},
		"zh-Hans": {1: "阔剑", 22: "长戟", 53: "钉头锤", 128: "布衣", 129: "旅行装", 132: "皮甲", 165: "法师袍", 192: "药草", 193: "回复剂"},
	}
	for localeID, entries := range wants {
		assertReviewedContentEntryIDs(t, localeID, entries)
		catalog, err := loadOfficialLocaleEntities(localeID)
		if err != nil {
			t.Fatal(err)
		}
		for itemID, want := range itemWants[localeID] {
			got, err := catalog.ItemName(itemID)
			if err != nil || got != want {
				t.Fatalf("%s item %d=%q err=%v, want %q", localeID, itemID, got, err, want)
			}
		}
	}
}

func assertReviewedContentEntryIDs(t *testing.T, localeID string, wants map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(assetPath("assets/locales/" + localeID + "/content.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Entries []struct {
			StringID string `json:"string_id"`
			Status   string `json:"status"`
			Text     string `json:"text"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	remaining := make(map[string]string, len(wants))
	for id, text := range wants {
		remaining[id] = text
	}
	for _, entry := range document.Entries {
		want, ok := remaining[entry.StringID]
		if !ok {
			continue
		}
		if entry.Status != "reviewed" || entry.Text != want {
			t.Fatalf("%s %s status=%q text=%q, want reviewed %q", localeID, entry.StringID, entry.Status, entry.Text, want)
		}
		delete(remaining, entry.StringID)
	}
	if len(remaining) != 0 {
		t.Fatalf("%s reviewed entries missing: %v", localeID, remaining)
	}
}

func assertReviewedContentEntries(t *testing.T, localeID string, wants map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(assetPath("assets/locales/" + localeID + "/content.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Entries []struct {
			StringID string `json:"string_id"`
			Status   string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	remaining := make(map[string]bool, len(wants))
	for lineID := range wants {
		remaining[lineID+"/text"] = true
	}
	for _, entry := range document.Entries {
		if !remaining[entry.StringID] {
			continue
		}
		if entry.Status != "reviewed" {
			t.Fatalf("%s %s status=%q", localeID, entry.StringID, entry.Status)
		}
		delete(remaining, entry.StringID)
	}
	if len(remaining) != 0 {
		t.Fatalf("%s reviewed entries missing: %v", localeID, remaining)
	}
}

func attachOfficialTestLocale(t *testing.T, g *Game, localeID string) {
	t.Helper()
	catalog, err := loadOfficialLocale(localeID)
	if err != nil {
		t.Fatal(err)
	}
	content, err := loadOfficialLocaleContent(localeID)
	if err != nil {
		t.Fatal(err)
	}
	g.localeID, g.localeCatalog, g.localeContent = localeID, catalog, content
}

func TestAllRuntimeStoriesMatchCanonicalLineIdentities(t *testing.T) {
	paths := assetGlob("assets/story/ch*.json")
	if len(paths) != 35 {
		t.Fatalf("story paths=%d", len(paths))
	}
	allLines := make([]campaign.Line, 0, 1564)
	for _, path := range paths {
		lines, err := loadStoryScriptWithIdentityAt(path, "", nil)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for index, line := range lines {
			if line.LineID == "" {
				t.Fatalf("%s line %d lacks canonical identity", path, index)
			}
		}
		allLines = append(allLines, lines...)
	}
	if len(allLines) != 1564 {
		t.Fatalf("canonical runtime story lines=%d, want 1564", len(allLines))
	}
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		for _, line := range allLines {
			if _, err := g.localizedStoryText(line); err != nil {
				t.Fatalf("%s %s: %v", localeID, line.LineID, err)
			}
		}
	}
}

func TestChapterOneStoryUsesAllOfficialContentCatalogs(t *testing.T) {
	sceneIndex := 0
	lines, err := loadStoryScriptWithIdentityAt("assets/story/ch01.json", "", &sceneIndex)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 || lines[0].LineID != "legacy/line/7ecb566a60db/scenes/0/lines/0" {
		t.Fatalf("first ch01 line identity = %#v", lines)
	}
	seen := make(map[string]string)
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatalf("load %s content: %v", localeID, err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		got, err := g.resolveCampaignDialogLine(lines[0], nil, nil)
		if err != nil {
			t.Fatalf("resolve %s story: %v", localeID, err)
		}
		want, err := content.StoryText(lines[0].LineID)
		if err != nil || got.Text != want {
			t.Fatalf("resolve %s text = %q, want %q (%v)", localeID, got.Text, want, err)
		}
		seen[localeID] = got.Text
	}
	if seen["zh-Hant"] == seen["en"] || seen["zh-Hant"] == seen["ja"] {
		t.Fatalf("chapter one first line did not use translated content: %#v", seen)
	}
}

func TestStoryLocaleFailsClosedForMissingIdentity(t *testing.T) {
	content, err := loadOfficialLocaleContent("en")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{localeID: "en", localeContent: content}
	if _, err := g.resolveCampaignDialogLine(campaign.Line{Speaker: 0, Text: "來源"}, nil, nil); err == nil {
		t.Fatal("story line without canonical identity was accepted")
	}
}

func TestBattleEmbeddedEventsUseOfficialStoryContent(t *testing.T) {
	seen := make(map[string][3]string)
	for _, localeID := range localeIDs {
		content, err := loadOfficialLocaleContent(localeID)
		if err != nil {
			t.Fatal(err)
		}
		g := &Game{localeID: localeID, localeContent: content}
		event61, ok := event61DialogueActions(g, 0, 10, 1)
		if !ok || len(event61) != 1 {
			t.Fatalf("%s event61 actions=%d err=%q", localeID, len(event61), g.loadErr)
		}
		event75, ok := event75DialogueActions(g, 0, &battle.Unit{BattleFig: 4, HasBattleFig: true})
		if !ok || len(event75) != 1 || event75[0].Speaker != 4 {
			t.Fatalf("%s event75 actions=%#v err=%q", localeID, event75, g.loadErr)
		}
		event76, ok := event76DialogueActions(g, 2)
		if !ok || len(event76) != 3 {
			t.Fatalf("%s event76 actions=%d err=%q", localeID, len(event76), g.loadErr)
		}
		for _, action := range append(append(event61, event75...), event76...) {
			if action.Text == "" {
				t.Fatalf("%s embedded event produced empty dialogue", localeID)
			}
		}
		seen[localeID] = [3]string{event61[0].Text, event75[0].Text, event76[0].Text}
	}
	if seen["zh-Hant"] == seen["en"] || seen["zh-Hant"] == seen["ja"] {
		t.Fatalf("embedded event dialogue did not use translated catalogs: %#v", seen)
	}
}
