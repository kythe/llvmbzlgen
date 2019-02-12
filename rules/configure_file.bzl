# Copyright 2019 The Kythe Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

def _configure_file(ctx):
    args = ctx.actions.args()
    if ctx.attr.strict:
        args.add("-strict")
    args.add("-outfile", ctx.outputs.out)
    args.add("-json", struct(**ctx.attr.defines).to_json())
    args.add(ctx.file.src)
    ctx.actions.run(
        outputs = [ctx.outputs.out],
        inputs = [ctx.file.src],
        executable = ctx.file._cmakedefines,
        arguments = [args],
        mnemonic = "ConfigureFile",
    )

configure_file = rule(
    implementation = _configure_file,
    doc = """Replace CMake-style template files with the variables defined in defines.

    For more details on the format, see https://cmake.org/cmake/help/latest/command/configure_file.html
    """,
    attrs = {
        "src": attr.label(
            allow_single_file = True,
            doc = "The template file on which to perform substitutions.",
        ),
        "out": attr.output(
            mandatory = True,
            doc = "The output file to write.",
        ),
        "strict": attr.bool(
            doc = "If true, exit with an error on undefined substitions.",
            default = False,
        ),
        "defines": attr.string_dict(
            doc = "A dictionary of substitutions to perform on the input file.",
        ),
        "_cmakedefines": attr.label(
            default = Label("@io_kythe_llvmbzlgen//tools/cmakedefines"),
            allow_single_file = True,
            executable = True,
            cfg = "host",
        ),
    },
)
