# Design

## Key flow

```marmaid
flowchart TD
    A[Input] --> B[global remaps]
    B --> C[layer def]
    C --> D[layer remaps]
```

```
[global]
remaps = { a = "b" }
[src]
def = "a | b | c"
[layer1]
def = "e | f | g"
remaps = { f = "x" }
```

```
| a | b | c |
      ↓
(global remaps)
    a -> b
| b | b | c
      ↓
  (layer def)
| f | f | g |
      ↓
(layer remaps)
    f -> x
| x | x | g |
```

