package moderation

import "testing"

// The word list is generated from a search-engine keyword collection
// (tools/lexicon), and collections like that one are full of ordinary
// vocabulary: 兼职, 自拍, 起诉, hardcore, triangle. Every hit holds a message
// for a human, so a word that ordinary chat uses is a word that costs an
// operator time. These two tests hold both directions of the line: the words
// below must stay quiet, and the messages below must still be caught.

var ordinaryChat = []string{
	"今天下午三点开会，别迟到",
	"这是一条完全正常的普通消息",
	"我妈让我周末回家吃饭",
	"北京的天气怎么样",
	"中央政府发布了新政策",
	"团建定在周五晚上",
	"在群里问一下团购的事",
	"公司章程我下午发你",
	"主权问题不讨论",
	"他在团里打鼓",
	"这个项目上线时间定了吗",
	"护照办好了吗",
	"帮我看看这段代码有没有 bug",
	"周末去爬山，你要一起吗",
	"苹果人还是安卓人",
	"我想买一个二手的相机",
	"房价最近降了吗",
	"公司官网是 china.com，可以看一下",
	"文档在 docs.google.com 上，链接我稍后发",
	"参考 https://www.baidu.com/s?wd=test 这个页面",
	"新版 App 明天上线，注意灰度",
	"这个接口的 response time 有点高",
	"孩子的作业写完了吗",
	"会议纪要我已经发到群里了",
	"这台服务器的 disk usage 到 80% 了",
	"天气转凉，记得加衣服",
	"我在淘宝上买了个键盘，明天到货",
	"这份工作是兼职，一周去两天",
	"周末去按摩放松一下",
	"他去医院检查了一下乳房，医生说没大事",
	"这家公司起诉了竞争对手",
	"他在公司做枪手写文案",
	"这个游戏太 hardcore 了",
	"我看了一部伦理片，讲家庭的",
	"先做 triangle 那部分的重构",
	"帮我把插头拔出来",
	"医生开了地塞米松，吃完好多了",
	"这条视频的铃声很好听",
	"买别墅的事情再说吧",
	"他八九不离十会同意",
	"公司独领风骚很多年了",
	"本周的排期我发你邮箱了",
	"我在楼下买了杯咖啡，咖啡因有点多",
}

var sensitiveChat = []string{
	"有人在群里发广告，还提到网络赌博，大家小心",
	"出售气枪和弓弩，需要的私聊",
	"加微信代开发票，假钞也有",
	"法轮功是邪教",
	"支持台独",
	"裸聊加我",
	"六四事件",
	"出售成品冰毒，量大从优",
	"买一把猎枪打猎",
	"出售仿真枪，支持货到付款",
	"加我微信卖原味内衣",
	"介绍一个六合彩网站给你",
	"办证刻章办，需要的联系",
	"海洛因现货",
	"组织线下援交活动",
	"这个站点 jx5178.3322.org 你懂的",
}

func TestDefaultListKeepsOrdinaryChatQuiet(t *testing.T) {
	matcher := NewMatcher(DefaultWords())
	for _, content := range ordinaryChat {
		if hits := matcher.Match(content); len(hits) != 0 {
			t.Errorf("ordinary message %q hit %v", content, hitsToWords(hits))
		}
	}
}

func TestDefaultListCatchesSensitiveChat(t *testing.T) {
	matcher := NewMatcher(DefaultWords())
	for _, content := range sensitiveChat {
		if hits := matcher.Match(content); len(hits) == 0 {
			t.Errorf("message %q hit nothing", content)
		}
	}
}
