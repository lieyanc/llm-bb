package engine

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"llm-bb/internal/config"
	"llm-bb/internal/llm"
	"llm-bb/internal/model"
	"llm-bb/internal/store"
)

type Engine struct {
	store  *store.Store
	client *llm.Client
	cfg    config.Config
	logger *log.Logger

	mu  sync.Mutex
	rng *rand.Rand
}

type Result struct {
	Message *model.Message
	Summary *model.Summary
	Skipped bool
	Reason  string
}

func New(store *store.Store, client *llm.Client, cfg config.Config, logger *log.Logger) *Engine {
	return &Engine{
		store:  store,
		client: client,
		cfg:    cfg,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) GenerateNextMessage(ctx context.Context, roomID int64) (Result, error) {
	room, err := e.store.GetRoom(ctx, roomID)
	if err != nil {
		return Result{}, err
	}
	if room.Status == model.RoomStatusPaused {
		return Result{Skipped: true, Reason: "room paused"}, nil
	}

	members, err := e.store.ListRoomMembers(ctx, roomID)
	if err != nil {
		return Result{}, err
	}
	if len(members) == 0 {
		return Result{Skipped: true, Reason: "room has no members"}, nil
	}

	recentDesc, err := e.store.ListRecentMessagesDescending(ctx, roomID, 18)
	if err != nil {
		return Result{}, err
	}

	if shouldSkipDenseOutput(room, recentDesc) {
		return Result{Skipped: true, Reason: "recent output too dense"}, nil
	}

	used, err := e.store.RoomTokenUsageToday(ctx, roomID)
	if err != nil {
		return Result{}, err
	}
	if room.DailyTokenBudget > 0 && used >= room.DailyTokenBudget {
		return Result{Skipped: true, Reason: "daily token budget reached"}, nil
	}

	personaIDs := make([]int64, 0, len(members))
	for _, member := range members {
		personaIDs = append(personaIDs, member.PersonaID)
	}
	relationships, err := e.store.ListRelationshipsForPersonas(ctx, personaIDs)
	if err != nil {
		return Result{}, err
	}
	relationMap := make(map[[2]int64]model.Relationship, len(relationships))
	for _, relationship := range relationships {
		relationMap[[2]int64{relationship.SourcePersonaID, relationship.TargetPersonaID}] = relationship
	}

	latestSummary, err := e.store.GetLatestSummary(ctx, roomID)
	if err != nil {
		return Result{}, err
	}

	selected, target, intent, replyTo, ok := e.selectSpeaker(room, members, recentDesc, relationMap)
	if !ok {
		return Result{Skipped: true, Reason: "no eligible speaker"}, nil
	}

	content, promptTokens, completionTokens := e.generateLine(ctx, room, selected, target, recentDesc, latestSummary, relationMap, intent)
	content = cleanGeneratedContent(selected.PersonaName, content)
	if content == "" {
		return Result{Skipped: true, Reason: "empty content after cleaning"}, nil
	}
	if isTooSimilar(content, recentDesc) {
		alt := cleanGeneratedContent(selected.PersonaName, e.composeFallbackLine(room, selected, target, intent, recentDesc))
		if alt == "" || isTooSimilar(alt, recentDesc) {
			return Result{Skipped: true, Reason: "generated content too similar"}, nil
		}
		content = alt
		promptTokens = 0
		completionTokens = estimateTokens(content)
	}

	message := &model.Message{
		RoomID:           room.ID,
		PersonaID:        selected.PersonaID,
		PersonaName:      selected.PersonaName,
		PersonaAvatar:    selected.Avatar,
		Kind:             model.MessageKindChat,
		Content:          content,
		ReplyToMessageID: replyTo,
		Source:           model.MessageSourceScheduler,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}
	if err := e.store.CreateMessage(ctx, message); err != nil {
		return Result{}, err
	}

	summary, err := e.maybeBuildSummary(ctx, room, latestSummary)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Message: message,
		Summary: summary,
	}, nil
}

func shouldSkipDenseOutput(room model.Room, recent []model.Message) bool {
	if len(recent) == 0 {
		return false
	}

	latest := recent[0]
	if latest.Kind == model.MessageKindUser {
		return false
	}

	minGap := time.Duration(room.TickMinSeconds) * time.Second / 2
	if minGap < 6*time.Second {
		minGap = 6 * time.Second
	}
	return time.Since(latest.CreatedAt) < minGap
}

func (e *Engine) selectSpeaker(
	room model.Room,
	members []model.RoomMemberView,
	recent []model.Message,
	relationMap map[[2]int64]model.Relationship,
) (model.RoomMemberView, *model.Message, string, int64, bool) {
	var lastMessage *model.Message
	if len(recent) > 0 {
		lastMessage = &recent[0]
	}

	lastByPersona := make(map[int64]time.Time)
	for _, message := range recent {
		if message.PersonaID > 0 {
			if _, ok := lastByPersona[message.PersonaID]; !ok {
				lastByPersona[message.PersonaID] = message.CreatedAt
			}
		}
	}

	mentionBoost := make(map[int64]int)
	var latestUser *model.Message
	for _, message := range recent {
		if message.Kind == model.MessageKindUser {
			latestUser = &message
			break
		}
	}
	if latestUser != nil {
		for _, member := range members {
			if strings.Contains(strings.ToLower(latestUser.Content), strings.ToLower("@"+member.PersonaName)) ||
				strings.Contains(strings.ToLower(latestUser.Content), strings.ToLower(member.PersonaName)) {
				mentionBoost[member.PersonaID] += 120
			}
		}
	}
	hasDirectMention := len(mentionBoost) > 0

	type candidate struct {
		member model.RoomMemberView
		weight float64
		intent string
		reply  int64
		named  bool
	}

	var candidates []candidate
	for _, member := range members {
		if !member.PersonaEnabled {
			continue
		}

		weight := float64(max(member.RoleWeight, 1) + member.ActivityLevel + room.Heat/2)
		replyTo := int64(0)
		intent := "抛出一个能推动群聊继续运转的新判断"

		named := false
		if mentioned := mentionBoost[member.PersonaID]; mentioned > 0 {
			named = true
			weight += float64(mentioned)
			if latestUser != nil {
				replyTo = latestUser.ID
				intent = "正面回应用户插话，同时保持角色口吻"
			}
		} else if hasDirectMention {
			weight *= 0.3
		}

		if lastMessage != nil {
			if lastMessage.PersonaID == member.PersonaID {
				weight *= 0.08
			} else if lastMessage.PersonaID > 0 {
				relation := relationMap[[2]int64{member.PersonaID, lastMessage.PersonaID}]
				weight += float64(relation.Affinity*2 + relation.Respect + relation.FocusWeight)
				weight += float64(relation.Hostility * max(room.ConflictLevel, 30) / 40)
				replyTo = lastMessage.ID

				switch {
				case relation.Hostility > relation.Affinity+15:
					intent = "针对上一条发言进行尖锐反驳或阴阳"
				case relation.Affinity > relation.Hostility+15:
					intent = "顺着上一条发言补刀、站队或放大观点"
				case room.ConflictLevel >= 65:
					intent = "围绕上一条发言继续拱火，让冲突再上一个台阶"
				default:
					intent = "对上一条发言提出不同角度，推动话题继续"
				}
			}
		}

		if lastAt, ok := lastByPersona[member.PersonaID]; ok {
			elapsed := time.Since(lastAt)
			cooldown := time.Duration(member.CooldownSeconds) * time.Second
			if cooldown <= 0 {
				cooldown = 20 * time.Second
			}
			if elapsed < cooldown {
				ratio := elapsed.Seconds() / cooldown.Seconds()
				if ratio < 0.25 {
					continue
				}
				weight *= math.Max(ratio, 0.15)
			}
		}

		if member.FactionID != 0 && room.ConflictLevel >= 70 && member.Aggression >= 60 {
			weight += 25
		}
		if !member.CanInitiate && replyTo == 0 {
			weight *= 0.35
		}
		if !member.CanReply && replyTo != 0 {
			weight *= 0.35
		}
		if weight <= 0 {
			continue
		}

		candidates = append(candidates, candidate{
			member: member,
			weight: weight,
			intent: intent,
			reply:  replyTo,
			named:  named,
		})
	}

	if len(candidates) == 0 {
		return model.RoomMemberView{}, nil, "", 0, false
	}

	if hasDirectMention {
		mentioned := make([]candidate, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.named {
				mentioned = append(mentioned, candidate)
			}
		}
		if len(mentioned) > 0 {
			candidates = mentioned
		}
	}

	selected := weightedPick(candidates, func(item candidate) float64 { return item.weight }, e.randomFloat64)

	var target *model.Message
	if selected.reply != 0 && lastMessage != nil && lastMessage.ID == selected.reply {
		target = lastMessage
	}

	return selected.member, target, selected.intent, selected.reply, true
}

func (e *Engine) generateLine(
	ctx context.Context,
	room model.Room,
	member model.RoomMemberView,
	target *model.Message,
	recentDesc []model.Message,
	latestSummary *model.Summary,
	relationMap map[[2]int64]model.Relationship,
	intent string,
) (content string, promptTokens int, completionTokens int) {
	if member.ProviderEnabled && strings.TrimSpace(member.ProviderBaseURL) != "" {
		messages := e.buildPrompt(room, member, target, recentDesc, latestSummary, relationMap, intent)
		modelName := strings.TrimSpace(member.ModelName)
		if modelName == "" {
			modelName = strings.TrimSpace(member.ProviderModel)
		}
		response, err := e.client.Complete(ctx, llm.ChatRequest{
			BaseURL:     member.ProviderBaseURL,
			APIKey:      member.ProviderAPIKey,
			Model:       modelName,
			Messages:    messages,
			Temperature: member.Temperature,
			MaxTokens:   member.MaxTokens,
			Timeout:     time.Duration(member.ProviderTimeoutMS) * time.Millisecond,
		})
		if err == nil && strings.TrimSpace(response.Content) != "" {
			return response.Content, response.PromptTokens, response.CompletionTokens
		}
		e.logger.Printf("llm fallback for room=%d persona=%s: %v", room.ID, member.PersonaName, err)
	}

	line := e.composeFallbackLine(room, member, target, intent, recentDesc)
	return line, 0, estimateTokens(line)
}

func (e *Engine) buildPrompt(
	room model.Room,
	member model.RoomMemberView,
	target *model.Message,
	recentDesc []model.Message,
	latestSummary *model.Summary,
	relationMap map[[2]int64]model.Relationship,
	intent string,
) []llm.Message {
	var recent strings.Builder
	chronological := slices.Clone(recentDesc)
	slices.Reverse(chronological)
	if len(chronological) > 10 {
		chronological = chronological[len(chronological)-10:]
	}
	for _, message := range chronological {
		speaker := "系统"
		if message.Kind == model.MessageKindUser {
			speaker = "观众"
		}
		if message.PersonaName != "" {
			speaker = message.PersonaName
		}
		recent.WriteString(fmt.Sprintf("[%s] %s\n", speaker, message.Content))
	}

	relationshipLine := "与你关系最强的对象：暂无明确关系。"
	if target != nil && target.PersonaID > 0 {
		if relation, ok := relationMap[[2]int64{member.PersonaID, target.PersonaID}]; ok {
			relationshipLine = fmt.Sprintf(
				"你对 %s 的关系：亲近=%d，敌意=%d，尊重=%d，关注=%d。备注：%s",
				target.PersonaName,
				relation.Affinity,
				relation.Hostility,
				relation.Respect,
				relation.FocusWeight,
				emptyToFallback(relation.Notes, "无"),
			)
		}
	}

	summaryText := "暂无历史摘要。"
	if latestSummary != nil && strings.TrimSpace(latestSummary.Content) != "" {
		summaryText = latestSummary.Content
	}

	systemPrompt := fmt.Sprintf(`你正在扮演一个持续运转的群聊背景板角色。

房间信息：
- 房间名：%s
- 主题：%s
- 风格：中文、短句、群聊口吻、适合持续观看
- 规则：一次只输出一条消息，不要输出角色名，不要加引号，不要写动作描写，不要写旁白，不要分点。

角色定义：
- 名字：%s
- 公开身份：%s
- 说话风格：%s
- 当前立场：%s
- 目标：%s
- 禁区：%s
- 阵营：%s
- 阵营描述：%s
- 攻击性：%d/100
- 活跃度：%d/100
- %s

对话摘要：
%s
`,
		room.Name,
		room.Topic,
		member.PersonaName,
		emptyToFallback(member.PublicIdentity, "无"),
		emptyToFallback(member.SpeakingStyle, "自然"),
		emptyToFallback(member.Stance, "未指定"),
		emptyToFallback(member.Goal, "推动话题继续"),
		emptyToFallback(member.Taboo, "无"),
		emptyToFallback(member.FactionName, "无阵营"),
		emptyToFallback(member.FactionDescription, "无"),
		member.Aggression,
		member.ActivityLevel,
		relationshipLine,
		summaryText,
	)

	userPrompt := fmt.Sprintf(`最近消息：
%s

本轮任务：
- %s
- 如果上一条在挑衅你，就接住。
- 如果用户点名你，就优先正面回应。
- 保持人物一致性和攻击性/克制程度。
- 控制在 16 到 70 个中文字符之间。
`, recent.String(), intent)

	return []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
}

func (e *Engine) composeFallbackLine(
	room model.Room,
	member model.RoomMemberView,
	target *model.Message,
	intent string,
	recentDesc []model.Message,
) string {
	var opener string
	switch {
	case strings.Contains(intent, "反驳"):
		openers := []string{"你这套话术也太省事了吧", "你这个结论下得比证据跑得还快", "别急着装已经定案了"}
		opener = openers[e.randomIntn(len(openers))]
	case strings.Contains(intent, "拱火"):
		openers := []string{"别收着了，问题根本没到能装平静的时候", "现在开始讲体面，未免太晚了", "你们都别急着下台，这戏还没演完"}
		opener = openers[e.randomIntn(len(openers))]
	case strings.Contains(intent, "回应用户"):
		openers := []string{"你这句插得很到位", "这话一进来，重点就清楚多了", "你提这个点，刚好把遮羞布扯开了"}
		opener = openers[e.randomIntn(len(openers))]
	default:
		openers := []string{"先别装看不见重点", "问题其实一直摆在这儿", "说白了，这事没那么复杂"}
		opener = openers[e.randomIntn(len(openers))]
	}

	angle := emptyToFallback(member.Stance, room.Topic)
	goal := emptyToFallback(member.Goal, "继续把话题往前推")
	styleTail := []string{
		fmt.Sprintf("我只提醒一句，%s。", angle),
		fmt.Sprintf("真要往下聊，就先承认大家都在围着 %s 打转。", angle),
		fmt.Sprintf("别绕了，核心就是 %s。", angle),
		fmt.Sprintf("你要是不肯碰这个前提，后面全是表演。"),
	}
	tail := styleTail[e.randomIntn(len(styleTail))]

	if member.Aggression >= 70 {
		tail = []string{
			fmt.Sprintf("继续装客观也没用，核心就是 %s。", angle),
			fmt.Sprintf("你不把 %s 摆上台面，后面全是糊弄。", angle),
			fmt.Sprintf("别打太极了，真正想护着的就是 %s。", angle),
		}[e.randomIntn(3)]
	}

	if target != nil && target.PersonaName != "" {
		return fmt.Sprintf("%s，%s。%s", target.PersonaName, opener, tail)
	}

	if len(recentDesc) > 0 && recentDesc[0].Kind == model.MessageKindUser {
		return fmt.Sprintf("%s，但我顺着你的问题说透一点：%s", opener, tail)
	}

	return fmt.Sprintf("%s。%s 顺便提醒一句，%s", opener, goal, tail)
}

func (e *Engine) maybeBuildSummary(ctx context.Context, room model.Room, latestSummary *model.Summary) (*model.Summary, error) {
	var summaryID int64
	if latestSummary != nil {
		summaryID = latestSummary.ID
	}

	count, err := e.store.CountMessagesSinceSummary(ctx, room.ID, summaryID)
	if err != nil {
		return nil, err
	}
	if count < room.SummaryTriggerCount {
		return nil, nil
	}

	messages, err := e.store.ListRoomMessages(ctx, room.ID, room.SummaryTriggerCount+10)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}

	var fromID int64 = messages[0].ID
	var toID int64 = messages[len(messages)-1].ID
	if latestSummary != nil {
		var filtered []model.Message
		for _, message := range messages {
			if message.ID > latestSummary.ToMessageID {
				filtered = append(filtered, message)
			}
		}
		if len(filtered) == 0 {
			return nil, nil
		}
		messages = filtered
		fromID = messages[0].ID
		toID = messages[len(messages)-1].ID
	}

	summary := &model.Summary{
		RoomID:        room.ID,
		FromMessageID: fromID,
		ToMessageID:   toID,
		Content:       buildSummaryContent(messages),
	}
	if err := e.store.CreateSummary(ctx, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func buildSummaryContent(messages []model.Message) string {
	if len(messages) == 0 {
		return "暂无新的对话摘要。"
	}

	speakers := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for _, message := range messages {
		if message.PersonaName == "" {
			continue
		}
		if _, ok := seen[message.PersonaName]; ok {
			continue
		}
		seen[message.PersonaName] = struct{}{}
		speakers = append(speakers, message.PersonaName)
		if len(speakers) == 4 {
			break
		}
	}

	last := messages[len(messages)-1]
	first := messages[0]

	return fmt.Sprintf(
		"最近一段对话主要围绕“%s”展开。参与较多的角色有：%s。对话从“%s”一路推进到“%s”，整体气氛偏%s，仍未解决的点是大家对同一问题的归因和立场分歧。",
		extractTopic(messages),
		emptyToFallback(strings.Join(speakers, "、"), "多名角色"),
		trimForSummary(first.Content),
		trimForSummary(last.Content),
		inferTone(messages),
	)
}

func extractTopic(messages []model.Message) string {
	for _, message := range messages {
		if message.Kind == model.MessageKindUser {
			return trimForSummary(message.Content)
		}
	}
	return trimForSummary(messages[len(messages)-1].Content)
}

func inferTone(messages []model.Message) string {
	score := 0
	for _, message := range messages {
		content := message.Content
		if strings.Contains(content, "？") || strings.Contains(content, "你") {
			score++
		}
		if strings.Contains(content, "别") || strings.Contains(content, "装") || strings.Contains(content, "问题") {
			score++
		}
	}
	if score >= len(messages) {
		return "对线"
	}
	return "拉扯"
}

func trimForSummary(content string) string {
	content = cleanWhitespace(content)
	if utf8.RuneCountInString(content) <= 22 {
		return content
	}
	runes := []rune(content)
	return string(runes[:22]) + "..."
}

func cleanGeneratedContent(name, content string) string {
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"'`")
	content = cleanWhitespace(content)

	for _, prefix := range []string{name + "：", name + ":"} {
		if strings.HasPrefix(content, prefix) {
			content = strings.TrimSpace(strings.TrimPrefix(content, prefix))
		}
	}

	runes := []rune(content)
	if len(runes) > 96 {
		content = string(runes[:96])
	}
	return strings.TrimSpace(content)
}

func cleanWhitespace(text string) string {
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}

func isTooSimilar(content string, recent []model.Message) bool {
	normalized := normalizeForSimilarity(content)
	for _, message := range recent {
		if normalizeForSimilarity(message.Content) == normalized {
			return true
		}
		if similarity(normalized, normalizeForSimilarity(message.Content)) >= 0.88 {
			return true
		}
	}
	return false
}

func normalizeForSimilarity(content string) string {
	replacer := strings.NewReplacer(" ", "", "，", "", "。", "", "！", "", "？", "", "、", "", "：", "", ":", "")
	return strings.ToLower(replacer.Replace(content))
}

func similarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	setA := make(map[rune]struct{})
	setB := make(map[rune]struct{})
	for _, r := range a {
		setA[r] = struct{}{}
	}
	for _, r := range b {
		setB[r] = struct{}{}
	}
	intersection := 0
	for r := range setA {
		if _, ok := setB[r]; ok {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1
	}
	return float64(intersection) / float64(union)
}

func estimateTokens(content string) int {
	length := utf8.RuneCountInString(content)
	return max(length/2, 12)
}

func weightedPick[T any](items []T, weightFn func(T) float64, randomFn func() float64) T {
	total := 0.0
	for _, item := range items {
		total += weightFn(item)
	}
	if total <= 0 {
		return items[0]
	}

	target := randomFn() * total
	seen := 0.0
	for _, item := range items {
		seen += weightFn(item)
		if target <= seen {
			return item
		}
	}
	return items[len(items)-1]
}

func (e *Engine) randomFloat64() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rng.Float64()
}

func (e *Engine) randomIntn(n int) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rng.Intn(n)
}

func emptyToFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
