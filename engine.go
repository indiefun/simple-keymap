package main

type rule struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Enabled bool   `json:"enabled"`
}

type output struct {
	Key  string `json:"key"`
	Down bool   `json:"down"`
}

type heldKey struct {
	target   string
	released bool
}

// Mapping state belongs to the window / hook thread; only outputs cross threads.
type engine struct {
	enabled bool
	rules   map[string]string
	held    map[string]heldKey
	refs    map[string]int
}

func newEngine() *engine {
	return &engine{rules: map[string]string{}, held: map[string]heldKey{}, refs: map[string]int{}}
}

// configure releases existing outputs before changing the rules. Keep swallowed
// sources until their physical key-up so a mid-press edit never leaks an orphan up.
func (e *engine) configure(enabled bool, rules []rule) []output {
	out := e.releaseAll()
	e.enabled = enabled
	e.rules = map[string]string{}
	for _, r := range rules {
		if r.Enabled {
			e.rules[r.Source] = r.Target
		}
	}
	return out
}

func (e *engine) releaseAll() []output {
	var out []output
	for target := range e.refs {
		out = append(out, output{Key: target})
	}
	clear(e.refs)
	for source, held := range e.held {
		held.released = true
		e.held[source] = held
	}
	return out
}

func (e *engine) process(source string, down bool) (bool, []output) {
	if held, ok := e.held[source]; ok {
		if down {
			k, _ := findKey(held.target)
			if !held.released && !k.Mouse {
				return true, []output{{Key: held.target, Down: true}}
			}
			return true, nil
		}
		delete(e.held, source)
		if !held.released {
			e.refs[held.target]--
			if e.refs[held.target] == 0 {
				delete(e.refs, held.target)
				return true, []output{{Key: held.target}}
			}
		}
		return true, nil
	}
	target, ok := e.rules[source]
	if !down || !e.enabled || !ok {
		return false, nil
	}
	e.held[source] = heldKey{target: target}
	e.refs[target]++
	if e.refs[target] == 1 {
		return true, []output{{Key: target, Down: true}}
	}
	return true, nil
}
