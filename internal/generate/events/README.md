# Generate events

Parse the markdown table of events located in the discord-api-docs sub module and extract each row. We then generate 
a list of events. This is expected to always work, every commit pushed to the main branch is checked for changed.

If the code generation fails as there were changes in the discord-api-docs markdown, we fix the parsing, instead of 
going back to handwriting the values. This is the current best effort to stay up to date and automatically detect
changes.
