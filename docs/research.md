# Design research

The earlier research described the retired 0.1 design: an MCP server, a Rex supervisor, restricted worker profiles, and parsed one-shot output. Those mechanisms were removed for 1.0, so their comparison and acceptance claims no longer describe the product.

[The current design](design.md) replaces that approach with upstream pstack workflows, a shared model sheet, and native or interactive backend delegation. [The delegation contract](../overlay/delegation.md) defines host behavior, [upstream maintenance](upstream.md) describes generation, and [status](status.md) separates live evidence from pending acceptance.
