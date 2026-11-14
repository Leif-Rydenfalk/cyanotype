# verify

`cyanotype verify <root>` builds the bom tree of `<root>` (same as `cyanotype tree`)
and runs checkers over it. Each checker returns results: `pass`, `fail`, `skip` or
`error`, each about one node path like `robot/arm/bracket`. The command exits non-zero
if anything failed or errored.

This is a first draft of the interface, meant to be iterated on. It only reads the
instantiated tree (`bomtree.Node` and `model.Item`), not contracts or coitems, so it
stays out of the way while those change.

## Go checkers

```go
type Checker interface {
	Name() string
	Check(ctx context.Context, in *verify.Input) ([]verify.Result, error)
}
```

Register with `verify.Register(c)`. `QuantityChecker` (every node has qty > 0) is the
example. An error from `Check` means the checker could not run; it is reported as an
`error` result and the other checkers still run.

## External checkers

Any program can be a checker:

```sh
cyanotype verify robot --exec fea="python3 fea_check.py"
```

It gets one JSON object on stdin:

```json
{
  "version": 1,
  "root": {
    "name": "robot",
    "path": "robot",
    "qty": 1,
    "item_qualifier": "robot",
    "item_digest": "…",
    "item": { "type": "…", "content": { "name": "…", "part_number": "…", "details": {} } },
    "children": [ { "name": "arm", "path": "robot/arm", "qty": 2, "children": [] } ]
  }
}
```

and writes one JSON object on stdout:

```json
{ "results": [ { "subject": "robot/arm", "status": "fail", "message": "max stress 310 MPa > 250 MPa" } ] }
```

A non-zero exit status means the checker itself failed (stderr becomes the message).

## Flags

- `-c, --checker name` run only these checkers (repeatable, default all)
- `--exec name=command` add an external checker
- `--json` print the report as JSON
- `--list` list checkers and exit
