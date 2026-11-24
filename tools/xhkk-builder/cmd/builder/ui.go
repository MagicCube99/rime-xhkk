package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"rime-xhkk-builder/pkg/rime"
)

// applyTheme configures modern Microsoft YaHei typography, Radix Slate palette, and crisp control metrics.
func applyTheme(c *ui.Context) {
	t := *c.Theme()
	t.Font = "Microsoft YaHei UI"
	t.Radius = 6

	if t.Dark {
		t.Background = ui.Hex("#0f172a") // Radix Slate-1
		t.Surface = ui.Hex("#1e293b")    // Radix Slate-2
		t.SurfaceHover = ui.Hex("#334155")
		t.SurfacePressed = ui.Hex("#475569")
		t.Border = ui.Hex("#334155")    // Radix Slate-6
		t.Text = ui.Hex("#f8fafc")      // Radix Slate-12
		t.TextMuted = ui.Hex("#94a3b8") // Radix Slate-11
		t.Accent = ui.Hex("#3b82f6")    // Radix Blue-9
	} else {
		t.Background = ui.Hex("#ffffff")
		t.Surface = ui.Hex("#f8fafc")      // Radix Slate-2
		t.SurfaceHover = ui.Hex("#f1f5f9") // Radix Slate-3
		t.SurfacePressed = ui.Hex("#e2e8f0")
		t.Border = ui.Hex("#e2e8f0")    // Radix Slate-6
		t.Text = ui.Hex("#0f172a")      // Radix Slate-12
		t.TextMuted = ui.Hex("#64748b") // Radix Slate-11
		t.Accent = ui.Hex("#2563eb")    // Radix Blue-9
	}
	c.SetTheme(&t)
}

// BuildUI renders the declarative user interface adhering to Radix typography and Scheme A split layout.
func BuildUI(c *ui.Context, s *AppState, win *mygo.Window) {
	applyTheme(c)
	s.Sync()

	trimmed := strings.TrimSpace(s.Word)
	isEmpty := (trimmed == "")

	// Window-level Enter key shortcut for quick addition or redeploy
	c.OnWindowShortcut(0, ui.KeyEnter, func() {
		if isEmpty {
			if s.RimeDirValid && s.DeployerReady {
				s.Redeploy(win)
			}
		} else if s.CanSubmit {
			s.Submit(win)
			s.InputHandle.RequestFocus(c)
		}
	})

	ui.Column(c).Padding(14, 16, 12, 16).Gap(9).Fill().Children(func() {
		// Form Area
		renderWordRow(c, s, win)
		renderCodeRow(c, s)
		renderPinyinRow(c, s)

		// Split Footer Area (Scheme A: Left 3 status lines, Right anchored action button)
		renderBottomSection(c, s, win)
	})
}

// renderWordRow renders Row 1: Word TextInput and Status Badge.
// The badge container is pinned to Width(110) matching the Segmented control on Row 2,
// guaranteeing that the Row 1 and Row 2 TextInput boxes have the exact same pixel width!
func renderWordRow(c *ui.Context, s *AppState, win *mygo.Window) {
	ui.Row(c).Height(32).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, "新词").FontSize(13).FontWeight(600).Width(34)

		in := ui.TextInput(c, &s.Word).Bind(&s.InputHandle).AutoFocus().Grow(1).Placeholder("输入词条 (多个用空格隔开)")
		if in.Changed() {
			s.SetWord(s.Word)
		}
		if in.Submitted() && s.CanSubmit {
			s.Submit(win)
			s.InputHandle.RequestFocus(c)
		}

		ui.Box(c).Width(110).AlignItems(ui.Center).Children(func() {
			renderStatusBadge(c, s)
		})
	})
}

// renderStatusBadge renders the inline colored status badge using Radix UI colors.
func renderStatusBadge(c *ui.Context, s *AppState) {
	t := c.Theme()
	var bg, fg ui.Color

	switch s.BadgeKind {
	case BadgeGreen:
		if t.Dark {
			bg = ui.RGBA(20, 83, 45, 0.35)
			fg = ui.Hex("#4ade80")
		} else {
			bg = ui.Hex("#dcfce7")
			fg = ui.Hex("#15803d")
		}
	case BadgeRed:
		if t.Dark {
			bg = ui.RGBA(127, 29, 29, 0.35)
			fg = ui.Hex("#f87171")
		} else {
			bg = ui.Hex("#fee2e2")
			fg = ui.Hex("#b91c1c")
		}
	case BadgeYellow:
		if t.Dark {
			bg = ui.RGBA(120, 53, 15, 0.35)
			fg = ui.Hex("#fbbf24")
		} else {
			bg = ui.Hex("#fef3c7")
			fg = ui.Hex("#b45309")
		}
	default: // BadgeGray
		if t.Dark {
			bg = ui.RGBA(30, 41, 59, 0.45)
			fg = ui.Hex("#94a3b8")
		} else {
			bg = ui.Hex("#f1f5f9")
			fg = ui.Hex("#475569")
		}
	}

	ui.Box(c).Padding(3, 8).Radius(12).Background(bg).Children(func() {
		ui.Text(c, s.BadgeText).FontSize(13).SingleLine().TextColor(fg).Bold().Tooltip(s.StatusMessage)
	})
}

// renderCodeRow renders Row 2: Derived code box and target dictionary segmented control.
func renderCodeRow(c *ui.Context, s *AppState) {
	t := c.Theme()
	ui.Row(c).Height(32).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, "编码").FontSize(13).FontWeight(600).Width(34)
		ui.TextInput(c, &s.DisplayCode).ReadOnly(true).Grow(1)

		if s.SurnameHint {
			hint := ui.Text(c, "💡 建议选人名").FontSize(11).TextColor(t.Accent).Cursor(ui.CursorPointer).SingleLine().Tooltip("检测到百家姓氏，点击可直接切换至“人名”词库以生成3码简码")
			if hint.Clicked() {
				s.SetTargetDict(TargetNames)
			}
		}

		selected := 0
		if s.TargetDict == TargetNames {
			selected = 1
		}
		ui.Box(c).Width(110).Children(func() {
			if ui.Segmented(c, &selected, "自用", "人名").FillWidth().Changed() {
				if selected == 1 {
					s.SetTargetDict(TargetNames)
				} else {
					s.SetTargetDict(TargetCustom)
				}
			}
		})
	})
}

// renderPinyinRow renders Row 3: Phoneme capsules or multi-word summary list.
// For 3-character person names, the optional 3-code checkbox is integrated cleanly on the right.
func renderPinyinRow(c *ui.Context, s *AppState) {
	ui.Row(c).Height(30).AlignItems(ui.Center).Gap(6).Children(func() {
		if s.IsBatch {
			ui.Text(c, "清单").FontSize(13).FontWeight(600).Width(34)
			for _, item := range s.BatchItems {
				t := c.Theme()
				var fg, bg ui.Color
				switch item.BadgeKind {
				case BadgeGreen:
					fg = t.Success
					bg = t.Success.Alpha(0.15)
				case BadgeRed:
					fg = t.Danger
					bg = t.Danger.Alpha(0.15)
				case BadgeYellow:
					fg = t.Warning
					bg = t.Warning.Alpha(0.15)
				default:
					fg = t.TextMuted
					bg = t.TextMuted.Alpha(0.15)
				}
				label := fmt.Sprintf("%s: %s", item.Word, item.DisplayCode)
				ui.Box(c).Padding(2, 6).Radius(4).Background(bg).Children(func() {
					ui.Text(c, label).FontSize(12).TextColor(fg)
				})
			}
			return
		}

		ui.Text(c, "读音").FontSize(13).FontWeight(600).Width(34)
		if len(s.CharStates) == 0 {
			ui.Text(c, "输入 2 字及以上中文自动显示各字读音").FontSize(12).TextColor(c.Theme().TextMuted)
			return
		}

		for i := range s.CharStates {
			cs := &s.CharStates[i]
			idx := i
			hasAlternates := len(cs.Readings) > 1

			if hasAlternates {
				// Clickable button indicating multiple readings available
				label := fmt.Sprintf("%c: %s ▾", cs.Char, cs.Flypy)
				btn := ui.Button(c, label).FontSize(12).Tooltip("多音字，点击切换读音")
				if btn.Clicked() {
					s.CycleReading(idx)
				}
			} else {
				// Standard static capsule
				label := fmt.Sprintf("%c: %s", cs.Char, cs.Flypy)
				t := c.Theme()
				ui.Box(c).Padding(2, 6).Radius(4).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Text(c, label).FontSize(12)
				})
			}
		}

		// Right-side dynamic options (3-code option and/or Top Override option)
		if !s.IsBatch && len(s.CharStates) > 0 {
			t := c.Theme()

			// Option A: 3-character person name 3-code option
			if s.HasShortCode {
				ui.Spacer(c).Grow(1)
				cb := ui.Checkbox(c, &s.IncludeShortCode, fmt.Sprintf("包含3码 (%s)", s.ShortCode))
				if !s.RimeDirValid {
					cb.Disabled(true)
				}
				if cb.Changed() && s.RimeDirValid {
					s.SetIncludeShortCode(s.IncludeShortCode)
				}

				if !s.RimeDirValid {
					ui.Text(c, "- 无法核验").FontSize(11).TextColor(t.TextMuted).SingleLine()
				} else if s.HasShortCodeCollision {
					ui.Text(c, fmt.Sprintf("⚠ 与'%s'重码", s.ShortCodeCollisionWord)).FontSize(11).TextColor(t.Warning).SingleLine()
				} else {
					ui.Text(c, "✓ 无重码").FontSize(11).TextColor(t.Success).SingleLine()
				}
			}

			// Option B: Top Override checkbox when collision occurs
			if s.CanTopOverride {
				if !s.HasShortCode {
					ui.Spacer(c).Grow(1)
				}
				cbTop := ui.Checkbox(c, &s.IsTopOverride, "强设为首选").Tooltip(fmt.Sprintf("勾选后空格直接上屏，原首选词‘%s’将降为次选", s.TopOverrideCollisionWord))
				if !s.RimeDirValid {
					cbTop.Disabled(true)
				}
				if cbTop.Changed() && s.RimeDirValid {
					s.SetTopOverride(s.IsTopOverride)
				}
			}
		}
	})
}

// renderBottomSection renders the split footer: left 3 primary metadata lines (13px bold labels), right anchored action button.
func renderBottomSection(c *ui.Context, s *AppState, win *mygo.Window) {
	t := c.Theme()
	ui.Column(c).Gap(6).FillWidth().Children(func() {
		ui.Divider(c)

		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
			// Left Column (Fixed min-width 290px so it can never be crushed)
			ui.Column(c).Grow(1).MinWidth(290).Gap(3).Children(func() {
				// Line 1: 用户文件夹
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, "用户文件夹:").FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()

					dispPath := s.RimeDir
					maxLen := 22
					if !s.RimeDirValid && s.DetectedRimeDirValid && s.DetectedRimeDir != "" {
						maxLen = 16
					}
					if len(dispPath) > maxLen {
						dispPath = "..." + dispPath[len(dispPath)-(maxLen-3):]
					}
					pathColor := t.Text
					if !s.RimeDirValid {
						pathColor = t.Danger
					}
					ui.Text(c, dispPath).FontSize(13).TextColor(pathColor).SingleLine().Tooltip(s.RimeDir)

					if !s.RimeDirValid {
						ui.Text(c, "[无效]").FontSize(13).TextColor(t.Danger).Bold().SingleLine().Tooltip(s.RimeDirError)

						if s.DetectedRimeDirValid && s.DetectedRimeDir != "" {
							restoreBtn := ui.Text(c, "还原 ↺").FontSize(13).FontWeight(500).TextColor(t.Accent).Cursor(ui.CursorPointer).SingleLine().Tooltip(fmt.Sprintf("还原为默认用户文件夹: %s", s.DetectedRimeDir))
							if restoreBtn.Hovered() {
								restoreBtn.Underline()
							}
							if restoreBtn.Clicked() {
								s.RestoreDetectedRimeDir()
							}
						}
					}

					changeBtn := ui.Text(c, "更换 ↗").FontSize(13).FontWeight(500).TextColor(t.Accent).Cursor(ui.CursorPointer).SingleLine().Tooltip("更换小鹤音形用户文件夹")
					if changeBtn.Hovered() {
						changeBtn.Underline()
					}
					if changeBtn.Clicked() {
						go func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
								Parent:      win,
								Title:       "选择小鹤音形用户文件夹 (必须包含 flypy.dict.yaml)",
								Directory:   true,
								DefaultPath: s.RimeDir,
							})
							if err == nil && len(paths) > 0 && paths[0] != "" {
								chosen := paths[0]
								valid, _ := rime.ValidateRimeDir(chosen)
								updateDir := func() {
									s.SetRimeDir(chosen)
									if !valid {
										s.DeployStatusMsg = "目录无效 (未找到小鹤方案)"
									} else {
										s.DeployStatusMsg = "已切换用户文件夹"
									}
								}
								if win != nil {
									win.Update(updateDir)
								} else {
									updateDir()
								}
							}
						}()
					}
				})

				// Line 2: 自定义词库 (三个纯文本超链接，13px 优雅排印)
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, "自定义词库:").FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()

					customColor := t.Accent
					customCursor := ui.CursorPointer
					customTip := "用默认编辑器打开 flypy.custom.dict.yaml 手工修改/删词"
					if !s.RimeDirValid {
						customColor = t.TextMuted
						customCursor = ui.CursorDefault
						customTip = "请先选择有效的小鹤音形用户文件夹"
					}
					customLink := ui.Text(c, "自用词库 ↗").FontSize(13).FontWeight(500).TextColor(customColor).Cursor(customCursor).SingleLine().Tooltip(customTip)
					if s.RimeDirValid && customLink.Hovered() {
						customLink.Underline()
					}
					if s.RimeDirValid && customLink.Clicked() {
						_ = s.OpenFile(TargetCustom)
					}

					namesColor := t.Accent
					namesCursor := ui.CursorPointer
					namesTip := "用默认编辑器打开 flypy.names.dict.yaml 手工修改/删词"
					if !s.RimeDirValid {
						namesColor = t.TextMuted
						namesCursor = ui.CursorDefault
						namesTip = "请先选择有效的小鹤音形用户文件夹"
					}
					namesLink := ui.Text(c, "人名词库 ↗").FontSize(13).FontWeight(500).TextColor(namesColor).Cursor(namesCursor).SingleLine().Tooltip(namesTip)
					if s.RimeDirValid && namesLink.Hovered() {
						namesLink.Underline()
					}
					if s.RimeDirValid && namesLink.Clicked() {
						_ = s.OpenFile(TargetNames)
					}

					topColor := t.Accent
					topCursor := ui.CursorPointer
					topTip := "用默认编辑器打开 flypy.user.top.dict.yaml 查看/管理置顶词"
					if !s.RimeDirValid {
						topColor = t.TextMuted
						topCursor = ui.CursorDefault
						topTip = "请先选择有效的小鹤音形用户文件夹"
					}
					topLink := ui.Text(c, "置顶词库 ↗").FontSize(13).FontWeight(500).TextColor(topColor).Cursor(topCursor).SingleLine().Tooltip(topTip)
					if s.RimeDirValid && topLink.Hovered() {
						topLink.Underline()
					}
					if s.RimeDirValid && topLink.Clicked() {
						_ = s.OpenFile(TargetUserTop)
					}
				})

				// Line 3: 小狼毫服务 (部署服务 + 算法服务双监控，13px 明确呈现)
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					statusText, allGood := s.WeaselServiceStatusText()
					statusColor := t.TextMuted
					if !allGood {
						statusColor = t.Warning
					}
					ui.Text(c, "小狼毫服务:").FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()
					ui.Text(c, statusText).FontSize(13).FontWeight(500).TextColor(statusColor).SingleLine()
				})
			})

			// Right Column: Fixed 155px width, action button & centered single-line message
			ui.Column(c).Width(130).AlignItems(ui.Center).Children(func() {
				trimmed := strings.TrimSpace(s.Word)
				isEmpty := (trimmed == "")
				btnLabel := s.SubmitButtonLabel()

				// Style 1: Fluent native neutral standard button (subtle border, non-distracting)
				btn := ui.Button(c, btnLabel).Width(130).Height(32).FontWeight(600)
				if (isEmpty && (!s.RimeDirValid || !s.DeployerReady)) || (!isEmpty && !s.CanSubmit) {
					btn.Disabled(true)
				}
				if btn.Clicked() {
					if isEmpty {
						if s.RimeDirValid && s.DeployerReady {
							s.Redeploy(win)
						}
					} else if s.CanSubmit {
						s.Submit(win)
						s.InputHandle.RequestFocus(c)
					}
				}

				if s.DeployStatusMsg != "" {
					msgColor := t.Accent
					if strings.Contains(s.DeployStatusMsg, "无效") || strings.Contains(s.DeployStatusMsg, "失败") || strings.Contains(s.DeployStatusMsg, "错误") {
						msgColor = t.Danger
					}
					ui.Text(c, s.DeployStatusMsg).FontSize(11).SingleLine().TextColor(msgColor).Bold().Margin(2, 0, 0, 0)
				}
			})
		})
	})
}
