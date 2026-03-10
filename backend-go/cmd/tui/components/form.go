package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FormHelper 表单辅助组件
type FormHelper struct {
	form      *tview.Form
	theme     *Theme
	onChanged func(field string, value string)
}

// NewFormHelper 创建新的表单辅助
func NewFormHelper(theme *Theme) *FormHelper {
	if theme == nil {
		theme = &DefaultTheme
	}

	form := tview.NewForm()
	form.SetBackgroundColor(theme.Background)
	form.SetFieldBackgroundColor(theme.Surface)
	form.SetFieldTextColor(theme.Text)
	form.SetLabelColor(theme.Primary)

	return &FormHelper{
		form:  form,
		theme: theme,
	}
}

// GetForm 获取底层表单
func (f *FormHelper) GetForm() *tview.Form {
	return f.form
}

// AddDropDown 添加下拉选择
func (f *FormHelper) AddDropDown(label string, options []string, initial int, onChanged func(string, string)) *FormHelper {
	f.form.AddDropDown(label, options, initial, func(option string, optionIndex int) {
		if onChanged != nil {
			onChanged(label, option)
		}
		if f.onChanged != nil {
			f.onChanged(label, option)
		}
	})
	return f
}

// AddInputField 添加输入框
func (f *FormHelper) AddInputField(label, value string, fieldWidth int, validator func(string, rune) bool, onChanged func(string, string)) *FormHelper {
	f.form.AddInputField(label, value, fieldWidth, validator, func(text string) {
		if onChanged != nil {
			onChanged(label, text)
		}
		if f.onChanged != nil {
			f.onChanged(label, text)
		}
	})
	return f
}

// AddPasswordField 添加密码输入框
func (f *FormHelper) AddPasswordField(label, value string, fieldWidth int, mask rune, onChanged func(string, string)) *FormHelper {
	f.form.AddPasswordField(label, value, fieldWidth, mask, func(text string) {
		if onChanged != nil {
			onChanged(label, text)
		}
		if f.onChanged != nil {
			f.onChanged(label, text)
		}
	})
	return f
}

// AddCheckbox 添加复选框
func (f *FormHelper) AddCheckbox(label string, checked bool, onChanged func(bool)) *FormHelper {
	f.form.AddCheckbox(label, checked, func(checked bool) {
		if onChanged != nil {
			onChanged(checked)
		}
		if f.onChanged != nil {
			f.onChanged(label, boolToString(checked))
		}
	})
	return f
}

// AddTextView 添加文本显示
func (f *FormHelper) AddTextView(label, text string, width, height int, wrap, scroll bool) *FormHelper {
	f.form.AddTextView(label, text, width, height, wrap, scroll)
	return f
}

// AddButton 添加按钮
func (f *FormHelper) AddButton(label string, onClicked func()) *FormHelper {
	btn := f.form.AddButton(label, onClicked)
	if btn != nil {
		btn.SetBackgroundColor(f.theme.Secondary)
		btn.SetLabelColor(f.theme.Text)
	}
	return f
}

// AddButtons 添加多个按钮
func (f *FormHelper) AddButtons(labels []string, onClicked func(int)) *FormHelper {
	for i, label := range labels {
		label := label
		index := i
		f.form.AddButton(label, func() {
			if onClicked != nil {
				onClicked(index)
			}
		})
	}
	return f
}

// SetOnChanged 设置变更回调
func (f *FormHelper) SetOnChanged(onChanged func(field string, value string)) *FormHelper {
	f.onChanged = onChanged
	return f
}

// SetButtonsAlign 设置按钮对齐方式
func (f *FormHelper) SetButtonsAlign(align int) *FormHelper {
	f.form.SetButtonsAlign(align)
	return f
}

// SetButtonBackgroundColor 设置按钮背景色
func (f *FormHelper) SetButtonBackgroundColor(color tcell.Color) *FormHelper {
	f.form.SetButtonBackgroundColor(color)
	return f
}

// SetButtonTextColor 设置按钮文字色
func (f *FormHelper) SetButtonTextColor(color tcell.Color) *FormHelper {
	f.form.SetButtonTextColor(color)
	return f
}

// SetBorder 设置边框
func (f *FormHelper) SetBorder(show bool) *FormHelper {
	f.form.SetBorder(show)
	return f
}

// SetTitle 设置标题
func (f *FormHelper) SetTitle(title string) *FormHelper {
	f.form.SetTitle(title)
	return f
}

// GetFormPrimitive 获取表单原始组件
func (f *FormHelper) GetFormPrimitive() tview.Primitive {
	return f.form
}

// FormWithCallbacks 创建带回调的表单
func FormWithCallbacks(theme *Theme, onChanged func(field string, value string)) *FormHelper {
	return NewFormHelper(theme).SetOnChanged(onChanged)
}

// boolToString 布尔转字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SettingsForm 设置页面表单
type SettingsForm struct {
	*FormHelper
	groups []FormGroup
}

// FormGroup 表单分组
type FormGroup struct {
	Title  string
	Fields []FormField
}

// FormField 表单项
type FormField struct {
	Label       string
	Type        string
	Value       string
	Options     []string
	OnChanged   func(string)
}

// NewSettingsForm 构建设置页面表单
func NewSettingsForm(theme *Theme) *SettingsForm {
	return &SettingsForm{
		FormHelper: NewFormHelper(theme),
		groups:     make([]FormGroup, 0),
	}
}

// AddGroup 添加分组
func (s *SettingsForm) AddGroup(title string) *SettingsForm {
	s.groups = append(s.groups, FormGroup{
		Title:  title,
		Fields: make([]FormField, 0),
	})
	return s
}

// AddGroupField 添加分组字段
func (s *SettingsForm) AddGroupField(field FormField) *SettingsForm {
	if len(s.groups) > 0 {
		s.groups[len(s.groups)-1].Fields = append(s.groups[len(s.groups)-1].Fields, field)
	}
	return s
}

// Build 构建表单
func (s *SettingsForm) Build() *tview.Form {
	// 分组标题可以通过 AddTextView 实现
	for _, group := range s.groups {
		// 添加分组标题
		s.form.AddTextView("", "[::b]"+group.Title, 40, 1, false, false)
		
		for _, field := range group.Fields {
			switch field.Type {
			case "dropdown":
				s.form.AddDropDown(field.Label, field.Options, 0, nil)
			case "input":
				s.form.AddInputField(field.Label, field.Value, 40, nil, nil)
			case "checkbox":
				s.form.AddCheckbox(field.Label, field.Value == "true", nil)
			}
		}
	}
	return s.form
}