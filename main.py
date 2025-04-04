import os
import pyedn
from pyedn import Keyword as Kw
from string import ascii_lowercase

# ======================
# CONFIGURATION
# ======================
TC_VARIABLE = "TC"  # Change this to modify the prefix in EDN rules
DEFAULT_KEY = " "    # What to show for unmapped keys

# ======================
# EDN PARSER
# ======================
def parse_edn_config(edn_path):
    """Parse EDN file and extract all key mappings"""
    with open(edn_path) as f:
        data = pyedn.load(f)
    
    config = {
        # Initialize all possible keys
        **{letter: DEFAULT_KEY for letter in ascii_lowercase},
        **{
            'open_bracket': DEFAULT_KEY,
            'close_bracket': DEFAULT_KEY,
            'semicolon': DEFAULT_KEY,
            'quote': DEFAULT_KEY,
            'backslash': DEFAULT_KEY,
            'comma': DEFAULT_KEY,
            'period': DEFAULT_KEY,
            'slash': DEFAULT_KEY,
            'backspace': 'BACK',
            'enter': 'ENTER',
            'right_shift': 'SHIFT',
            'right_option': 'ALT',
            'right_command': 'CMD',
            'spacebar': 'SPACE'
        }
    }
    
    tc_prefix = f":!{TC_VARIABLE}#P"
    
    for rule in data.get(Kw(":rules"), []):
        if not isinstance(rule, list) or len(rule) < 2:
            continue
            
        key, value = rule[0], rule[1]
        key_str = str(key)
        
        # Handle letter keys (a-z)
        for letter in ascii_lowercase:
            if key_str == f"{tc_prefix}{letter}":
                config[letter] = " ".join(value) if isinstance(value, list) else str(value)
                break
        
        # Special keys
        if key_str == f"{tc_prefix}open_bracket":
            config['open_bracket'] = " ".join(value)
        elif key_str == f"{tc_prefix}close_bracket":
            config['close_bracket'] = " ".join(value)
        # Add other special key conditions...
        elif key_str == f":!{TC_VARIABLE}left_command":
            config['left_command'] = " ".join(value)
        # Add remaining key mappings...
    
    return config

# ======================
# MARKDOWN GENERATOR
# ======================
def generate_markdown(config):
    """Generate keyboard layout with current config"""
    return f"""
```markdown
┌─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬───────────┐
| ~ ` | ! 1 | @ 2 | # 3 | $ 4 | % 5 | ^ 6 | & 7 | * 8 | ( 9 | ) 0 | _ - | + = | {config['backspace'].center(8)} |
| TAB | {config['q'].center(3)} | {config['w'].center(3)} | {config['e'].center(3)} | {config['r'].center(3)} | {config['t'].center(3)} | {config['y'].center(3)} | {config['u'].center(3)} | {config['i'].center(3)} | {config['o'].center(3)} | {config['p'].center(3)} | {config['open_bracket']} | {config['close_bracket']} | {config['backslash'].center(8)} |
| CAPS | {config['a'].center(3)} | {config['s'].center(3)} | {config['d'].center(3)} | {config['f'].center(3)} | {config['g'].center(3)} | {config['h'].center(3)} | {config['j'].center(3)} | {config['k'].center(3)} | {config['l'].center(3)} | {config['semicolon']} | {config['quote']} |      {config['enter'].center(8)}      |
| SHIFT  | {config['z'].center(3)} | {config['x'].center(3)} | {config['c'].center(3)} | {config['v'].center(3)} | {config['b'].center(3)} | {config['n'].center(3)} | {config['m'].center(3)} | {config['comma']} | {config['period']} | {config['slash']} |     {config['right_shift'].center(8)}     |
| CTRL | ALT | CMD │               {config['spacebar'].center(16)}               │ {config['right_command']} | {config['right_option']} │
└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴───────────┘
