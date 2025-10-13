#!/usr/bin/env python3
"""
Example vulnerable Python code for testing
"""

import os
import pickle

# Vulnerable: eval() usage
def process_user_input(user_input):
    result = eval(user_input)  # CVE-2024-TEST-004
    return result

# Vulnerable: pickle.loads() from untrusted source
def load_data(data):
    obj = pickle.loads(data)  # CVE-2024-TEST-005
    return obj

# Vulnerable: command injection
def run_command(user_input):
    os.system(f"echo {user_input}")  # CVE-2024-TEST-006

# Vulnerable: hardcoded credentials
DATABASE_PASSWORD = "supersecret123"  # CVE-2024-TEST-007
API_KEY = "sk-1234567890abcdef"

if __name__ == "__main__":
    print("Running vulnerable code...")

