package fees

import (
	"slices"
)

func (r Rule) matches(q Query) bool {
	if r.Method != q.Method || r.Currency != q.Currency || !r.ActiveAt(q.At) {
		return false
	}
	if r.Connector != nil && *r.Connector != q.Connector {
		return false
	}
	if r.CardType != nil && *r.CardType != q.CardType {
		return false
	}
	if r.Region != nil && *r.Region != q.Region {
		return false
	}
	return true
}

// resolve picks the single most specific active rule; a tie at the top is an AmbiguousRuleError.
func resolve(candidates []Rule, q Query) (Rule, error) {
	best := Specificity(-1)
	var top []Rule
	for _, r := range candidates {
		if !r.matches(q) {
			continue
		}
		switch s := r.Specificity(); {
		case s > best:
			best, top = s, []Rule{r}
		case s == best:
			top = append(top, r)
		}
	}
	switch len(top) {
	case 0:
		return Rule{}, ErrNoRule
	case 1:
		return top[0], nil
	}
	ids := make([]uint, len(top))
	for i, r := range top {
		ids[i] = r.ID
	}
	slices.Sort(ids)
	return Rule{}, &AmbiguousRuleError{Specificity: best, RuleIDs: ids}
}
