def _ignore(ctx, *args, **kwargs):
    pass

def _lb_library(
        ctx,
        name,
        parent,
        dependencies = [],
        library_name = None,
        required_libraries = [],
        add_to_library_groups = []):
    if library_name == None:
        library_name = name
    lib = _add_unique_entity(ctx, name, struct(
        library_name = library_name,
        deps = dependencies + required_libraries,
    ))
    if library_name != name:
        # We need to support lookups by library_name as well, so just alias them.
        _add_unique_entity(ctx, library_name, lib)

def _lb_tool(
        ctx,
        name,
        parent,
        dependencies = [],
        required_libraries = []):
    _add_unique_entity(ctx, name, struct(
        deps = dependencies + required_libraries,
    ))

def _add_unique_entity(ctx, name, value):
    if name in ctx._graph:
        fail("{} already exists in graph!".format(name))
    ctx._graph[name] = value
    return value

def library_dependencies(ctx, name):
    """Returns a list of library-named direct dependencies for name."""
    return [
        # Because we can't recurse in Starlark, we rely on the fact that
        # Library and Tool targets only have Library dependencies in LLVMBuild.
        ctx._graph[d].library_name
        for d in ctx._graph[name].deps
    ]

def make_context():
    """Returns a context object suitable for passing to a generated_llvm_build_targets macro."""
    return struct(
        _graph = {},
        group = _ignore,
        library = _lb_library,
        optional_library = _lb_library,
        library_group = _ignore,
        target_group = _ignore,
        tool = _lb_tool,
        build_tool = _lb_tool,
    )

# Exported struct with other public entities.
llvmbuild = struct(
    make_context = make_context,
    library_dependencies = library_dependencies,
)
