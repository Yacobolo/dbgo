# Test macros for verifying macro system works correctly

def upper(column):
    """Wrap column in UPPER()"""
    return "UPPER({})".format(column)

def coalesce(column, default):
    """Wrap column in COALESCE"""
    return "COALESCE({}, {})".format(column, default)

def safe_cast(column, dtype):
    """Generate a CAST expression"""
    return "CAST({} AS {})".format(column, dtype)
