package ansi

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
)

// A LinkElement is used to render hyperlinks.
type LinkElement struct {
	BaseURL  string
	URL      string
	Children []ElementRenderer
	SkipText bool
	SkipHref bool

	hyperlink, resetHyperlink string
	validURL                  bool
}

// Render renders a LinkElement.
func (e *LinkElement) Render(w io.Writer, ctx RenderContext) error {
	// Make OSC 8 hyperlink token.
	e.hyperlink, e.resetHyperlink, e.validURL = makeHyperlink(e.URL)

	if !e.SkipText {
		if err := e.renderTextPart(w, ctx); err != nil {
			return err
		}
	}
	if !e.SkipHref {
		if err := e.renderHrefPart(w, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (e *LinkElement) renderTextPart(w io.Writer, ctx RenderContext) error {
	for _, child := range e.Children {
		if r, ok := child.(StyleOverriderElementRenderer); ok { //nolint:nestif
			var b bytes.Buffer
			st := ctx.options.Styles.LinkText
			if err := r.StyleOverrideRender(&b, ctx, st); err != nil {
				return fmt.Errorf("glamour: error rendering with style: %w", err)
			}

			token := e.hyperlink + b.String() + e.resetHyperlink
			if _, err := io.WriteString(w, token); err != nil {
				return fmt.Errorf("glamour: error writing hyperlink: %w", err)
			}
		} else {
			var b bytes.Buffer
			if err := child.Render(&b, ctx); err != nil {
				return fmt.Errorf("glamour: error rendering: %w", err)
			}
			token := e.hyperlink + b.String() + e.resetHyperlink
			el := &BaseElement{
				Token: token,
				Style: ctx.options.Styles.LinkText,
			}
			if err := el.Render(w, ctx); err != nil {
				return fmt.Errorf("glamour: error rendering: %w", err)
			}
		}
	}
	return nil
}

func (e *LinkElement) renderHrefPart(w io.Writer, ctx RenderContext) error {
	prefix := ""
	if !e.SkipText {
		prefix = " "
	}

	if e.validURL {
		token := e.hyperlink + resolveRelativeURL(e.BaseURL, e.URL) + e.resetHyperlink
		el := &BaseElement{
			Token:  token,
			Prefix: prefix,
			Style:  ctx.options.Styles.Link,
		}
		if err := el.Render(w, ctx); err != nil {
			return err
		}
	}
	return nil
}

// makeHyperlink takes a URL and returns an OSC 8 hyperlink token.
//
// 로컬 패치: OSC 8 을 내지 않는다 (빈 토큰을 돌려준다). URL 은 본문에 그대로 찍히므로
// 링크를 잃지 않고, 상대경로·원격 세션에서는 클릭해도 열 곳이 없다. 또 퍼센트 인코딩하지
// 않은 한글 URL 을 줄바꿈이 OSC 시퀀스 중간(UTF-8 바이트 사이)에서 잘라 글자가 화면에 샜다.
func makeHyperlink(link string) (string, string, bool) {
	u, err := url.Parse(link)
	validURL := err == nil && "#"+u.Fragment != link // if the URL only consists of an anchor, ignore it
	return "", "", validURL
}
