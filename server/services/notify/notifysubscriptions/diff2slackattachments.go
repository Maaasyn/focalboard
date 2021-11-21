// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package notifysubscriptions

import (
	"bytes"
	"fmt"

	"github.com/mattermost/focalboard/server/model"
	"github.com/wiggin77/merror"

	mm_model "github.com/mattermost/mattermost-server/v6/model"
)

// Diffs2SlackAttachments converts a slice of `Diff` to slack attachments to be used in a post.
func Diffs2SlackAttachments(diffs []*Diff, opts MarkdownOpts) ([]*mm_model.SlackAttachment, error) {
	var attachments []*mm_model.SlackAttachment
	merr := merror.New()

	for _, d := range diffs {
		// only handle cards for now.
		if d.BlockType == model.TypeCard {
			a, err := cardDiff2SlackAttachment(d, opts)
			if err != nil {
				merr.Append(err)
				continue
			}
			attachments = append(attachments, a)
		}
	}
	return attachments, merr.ErrorOrNil()
}

func cardDiff2SlackAttachment(cardDiff *Diff, opts MarkdownOpts) (*mm_model.SlackAttachment, error) {
	// sanity check
	if cardDiff.NewBlock == nil && cardDiff.OldBlock == nil {
		return nil, nil
	}

	attachment := &mm_model.SlackAttachment{}
	buf := &bytes.Buffer{}

	// card added
	if cardDiff.NewBlock != nil && cardDiff.OldBlock == nil {
		if err := execTemplate(buf, "AddCardNotify", opts, defAddCardNotify, cardDiff); err != nil {
			return nil, err
		}
		attachment.Pretext = buf.String()
		attachment.Fallback = attachment.Pretext
		return attachment, nil
	}

	// card deleted
	if cardDiff.NewBlock == nil && cardDiff.OldBlock != nil {
		buf.Reset()
		if err := execTemplate(buf, "DeleteCardNotify", opts, defDeleteCardNotify, cardDiff); err != nil {
			return nil, err
		}
		attachment.Pretext = buf.String()
		attachment.Fallback = attachment.Pretext
		return attachment, nil
	}

	// at this point new and old block are non-nil

	buf.Reset()
	if err := execTemplate(buf, "ModifyCardNotify", opts, defModifyCardNotify, cardDiff); err != nil {
		return nil, fmt.Errorf("cannot write notification for card %s: %w", cardDiff.NewBlock.ID, err)
	}
	attachment.Pretext = buf.String()
	attachment.Fallback = attachment.Pretext

	// title changes
	if cardDiff.NewBlock.Title != cardDiff.OldBlock.Title {
		attachment.Fields = append(attachment.Fields, &mm_model.SlackAttachmentField{
			Short: false,
			Title: "Title",
			Value: fmt.Sprintf("%s  ~~`%s`~~", cardDiff.NewBlock.Title, cardDiff.OldBlock.Title),
		})
	}

	// property changes
	if len(cardDiff.PropDiffs) > 0 {
		for _, propDiff := range cardDiff.PropDiffs {
			if propDiff.NewValue == propDiff.OldValue {
				continue
			}
			attachment.Fields = append(attachment.Fields, &mm_model.SlackAttachmentField{
				Short: false,
				Title: propDiff.Name,
				Value: fmt.Sprintf("%s  ~~`%s`~~", propDiff.NewValue, propDiff.OldValue),
			})
		}
	}

	// comment add/delete
	for _, child := range cardDiff.Diffs {
		if child.BlockType == model.TypeComment {
			var format string
			var block *model.Block
			if child.NewBlock != nil && child.OldBlock == nil {
				// added comment
				format = "%s"
				block = child.NewBlock
			}

			if child.NewBlock == nil && child.OldBlock != nil {
				// deleted comment
				format = "~~`%s`~~"
				block = child.OldBlock
			}

			if format != "" {
				attachment.Fields = append(attachment.Fields, &mm_model.SlackAttachmentField{
					Short: false,
					Title: "Comment",
					Value: fmt.Sprintf(format, stripNewlines(block.Title)),
				})
			}
		}
	}

	// content/description changes
	for _, child := range cardDiff.Diffs {
		if child.BlockType != model.TypeComment {
			if child.NewBlock.Title == child.OldBlock.Title {
				continue
			}

			/*
				TODO: use diff lib for content changes which can be many paragraphs.
				      Unfortunately `github.com/sergi/go-diff` is not suitable for
					  markdown display. An alternate markdown friendly lib is being
					  worked on at github.com/wiggin77/go-difflib and will be substituted
					  here when ready.

				newTxt := cleanBlockTitle(child.NewBlock)
				oldTxt := cleanBlockTitle(child.OldBlock)

				dmp := diffmatchpatch.New()
				txtDiffs := dmp.DiffMain(oldTxt, newTxt, true)

				_, _ = w.Write([]byte(dmp.DiffPrettyText(txtDiffs)))

			*/

			var newVal, oldVal string
			newVal = stripNewlines(child.NewBlock.Title)
			if child.OldBlock != nil && child.OldBlock.Title != "" {
				oldVal = fmt.Sprintf("\n\n~~`%s`~~\n", stripNewlines(child.OldBlock.Title))
			}

			attachment.Fields = append(attachment.Fields, &mm_model.SlackAttachmentField{
				Short: false,
				Title: "Comment",
				Value: newVal + oldVal,
			})
		}
	}
	return attachment, nil
}