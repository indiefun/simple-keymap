//go:build windows

package main

import (
	"errors"
	"unsafe"
)

func (a *application) selectedRule() int {
	return int(int32(send(a.controls[idList], lvmFirst+12, ^uintptr(0), 0x2)))
}

func (a *application) updateRuleActions() {
	selected := a.selectedRule()
	enabled := selected >= 0 && selected < len(a.cfg.Rules) && !a.editing && a.capturing == 0
	for _, id := range []int{idEdit, idDelete} {
		enableWindow.Call(a.controls[id], boolArg(enabled))
	}
}

func (a *application) refreshRuleStatuses() {
	previous := a.updating
	a.updating = true
	defer func() { a.updating = previous }()
	for index, r := range a.cfg.Rules {
		item := listItem{SubItem: 2, Text: utf16(a.ruleStateLabel(r))}
		send(a.controls[idList], lvmFirst+116, uintptr(index), uintptr(unsafe.Pointer(&item)))
	}
}

func (a *application) selectRule(index int) {
	item := listItem{State: 0x3, StateMask: 0x3}
	send(a.controls[idList], lvmFirst+43, uintptr(index), uintptr(unsafe.Pointer(&item)))
	send(a.controls[idList], lvmFirst+19, uintptr(index), 0) // LVM_ENSUREVISIBLE.
	a.updateRuleActions()
}

func (a *application) conflictingRule() int {
	i := int32(send(a.controls[idSource], cbGetCurSel, 0, 0))
	if i >= 0 && int(i) < len(keys) {
		for index, r := range a.cfg.Rules {
			if (!a.editing || index != a.editIndex) && r.Source == keys[i].ID {
				return index
			}
		}
	}
	return -1
}

func (a *application) locateDraftRule() {
	if index := a.conflictingRule(); index >= 0 {
		a.selectRule(index)
		a.status = "已选中已有规则，点击「编辑所选」可修改。"
		setFocus.Call(a.controls[idEdit])
		invalidateRect.Call(a.hwnd, 0, 0)
	}
}

func (a *application) beginEditSelected() {
	index := a.selectedRule()
	if a.editing || a.capturing != 0 || index < 0 || index >= len(a.cfg.Rules) {
		return
	}
	a.draftSource = send(a.controls[idSource], cbGetCurSel, 0, 0)
	a.draftTarget = send(a.controls[idTarget], cbGetCurSel, 0, 0)
	a.editing, a.editIndex = true, index
	for _, choice := range []struct {
		control int
		key     string
	}{{idSource, a.cfg.Rules[index].Source}, {idTarget, a.cfg.Rules[index].Target}} {
		for i, k := range keys {
			if k.ID == choice.key {
				send(a.controls[choice.control], cbSetCurSel, uintptr(i), 0)
				break
			}
		}
	}
	a.status = "正在修改所选规则；保存前，原规则继续保留。"
	a.updateRuleActions()
	a.updateDraft()
	setFocus.Call(a.controls[idTarget])
}

func (a *application) cancelEdit() {
	if !a.editing {
		return
	}
	a.editing = false
	send(a.controls[idSource], cbSetCurSel, a.draftSource, 0)
	send(a.controls[idTarget], cbSetCurSel, a.draftTarget, 0)
	a.status = "已取消修改，原规则保持不变。"
	a.updateRuleActions()
	a.updateDraft()
}

func (a *application) saveDraft() {
	if _, valid := a.draftHint(); !valid || a.capturing != 0 {
		return
	}
	i := send(a.controls[idSource], cbGetCurSel, 0, 0)
	j := send(a.controls[idTarget], cbGetCurSel, 0, 0)
	r := rule{Source: keys[i].ID, Target: keys[j].ID, Enabled: true}
	rules := append([]rule{}, a.cfg.Rules...)
	index := len(rules)
	message := "已添加："
	if a.editing {
		index = a.editIndex
		if index < 0 || index >= len(rules) {
			return
		}
		r.Enabled = rules[index].Enabled
		rules[index] = r
		message = "已修改："
	} else {
		rules = append(rules, r)
	}
	if err := validateRules(rules); err != nil {
		return
	}
	a.cfg.Rules = rules
	if a.editing {
		a.cancelEdit()
	}
	a.changed(message + keyCaption(r.Source) + " → " + keyCaption(r.Target) + "。")
	a.refreshList()
	a.selectRule(index)
}

func (a *application) persistConfig() {
	previousError := a.configSaveError
	a.configSaveError = ""
	if err := saveConfig(a.path, a.cfg); err != nil {
		// Report the cause without copying user names or file paths to diagnostics.
		for errors.Unwrap(err) != nil {
			err = errors.Unwrap(err)
		}
		a.configSaveError = err.Error()
	} else if previousError != "" {
		a.status = "保存已恢复，当前设置已保存到本机。"
	}
	showWindow.Call(a.controls[idRetrySave], boolArg(a.configSaveError != "")*5)
	if a.info != nil && a.info.page == "support" {
		a.updateInformation("support")
	}
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *application) footerCopy() string {
	if a.configSaveError != "" {
		return "更改未保存，重启后可能丢失。"
	}
	if a.inputFailureMessage != "" {
		return "输入已暂停，点击上方查看原因。"
	}
	if a.capturing != 0 {
		return "录入时暂停映射，Esc 取消。"
	}
	if a.status != "" {
		return a.status
	}
	return "所有更改自动保存"
}
