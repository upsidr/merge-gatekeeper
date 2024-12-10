import { LineBreakType, getLineBreak } from "./linebreak.js";
import { parse } from "yaml";
import { readFileSync } from "fs";
import replaceInFile from "replace-in-file";
import pkg from "showdown";
const { Converter } = pkg;
const converter = new Converter();

export interface Options {
  tocLevel?: number;
  sourceFile?: string;
  updateReadme?: boolean;
  readmeFile?: string;
  lineBreaks?: LineBreakType;
  includeNameHeader?: boolean;
}

interface YmlStructure {
  name: string;
  description: string;
  on: Record<string, WorkflowTriggerEvent>;
  inputs: ActionInputsOutputs;
  outputs: ActionInputsOutputs;
  runs: RunType;
}

interface RunType {
  using: string;
  main: string;
}

interface WorkflowTriggerEvent {
  types: string[];
  branches: string[];
  cron: string[];
  inputs: ActionInputsOutputs;
  secrets: ActionInputsOutputs;
  outputs: ActionInputsOutputs;
}

interface DefaultOptions {
  tocLevel: number;
  sourceFile: string;
  updateReadme: boolean;
  readmeFile: string;
  lineBreaks: LineBreakType;
  includeNameHeader: boolean;
}

export const defaultOptions: DefaultOptions = {
  tocLevel: 2,
  sourceFile: "action.yml",
  updateReadme: false,
  readmeFile: "README.md",
  lineBreaks: "LF",
  includeNameHeader: false,
};

type ActionInputsOutputs = Record<string, InputOutput>;

enum InputType {
  number,
  string,
  boolean,
}

enum InputOutputType {
  actionInput,
  workflowInput,
  actionOutput,
  workflowSecret,
}

const inputOutputHeaders: Record<InputOutputType, string[]> = {
  [InputOutputType.actionInput]: ["name", "description", "required", "default"],
  [InputOutputType.workflowInput]: [
    "name",
    "description",
    "type",
    "required",
    "default",
  ],
  [InputOutputType.actionOutput]: ["name", "description"],
  [InputOutputType.workflowSecret]: ["name", "description", "required"],
};

const inputOutputDefaults: Record<string, string> = {
  description: "",
  type: "",
  required: "false",
  default: '""',
};

interface InputOutput {
  required?: boolean;
  description?: string;
  default?: string;
  type?: InputType;
  deprecationMessage?: string;
}

function createMdTable(
  data: ActionInputsOutputs,
  options: DefaultOptions,
  type: InputOutputType,
): string {
  const tableData = getInputOutput(data, type);

  const headers = tableData.headers;
  const filler = Array(tableData.headers.length).fill("---");

  const result = [headers, filler]
    .concat(tableData.rows)
    .filter((x) => x.length > 0)
    .map((x) => `| ${x.join(" | ")} |${getLineBreak(options.lineBreaks)}`)
    .join("");

  return result;
}

function createMdCodeBlock(
  data: ActionInputsOutputs,
  options: DefaultOptions,
  isAction = true,
): string {
  let codeBlockArray = ["```yaml"];

  let indent = "";

  if (isAction) {
    codeBlockArray.push("- uses: ***PROJECT***@***VERSION***");
    indent += "  ";
  } else {
    codeBlockArray.push("jobs:");
    indent += "  ";
    codeBlockArray.push(`${indent}job1:`);
    indent += "  ";
    codeBlockArray.push(`${indent}uses: ***PROJECT***@***VERSION***`);
  }

  const inputs = getInputOutput(
    data,
    isAction ? InputOutputType.actionInput : InputOutputType.workflowInput,
    false,
  );

  if (data) {
    codeBlockArray.push(`${indent}with:`);
    indent += "  ";

    for (const row of inputs.rows) {
      const inputName = row[0];
      const inputDescCommented = row[1]
        .split(/(\r\n|\n|\r)/gm)
        .filter((l) => !["", "\r", "\n", "\r\n"].includes(l))
        .map((l) => `# ${l}`);
      const type = isAction ? undefined : row[2];
      const isRequired = isAction ? row[2] : row[3];
      const defaultVal = isAction ? row[3] : row[4];

      const inputBlock = [`${inputName}:`];
      inputBlock.push(...inputDescCommented);
      inputBlock.push("#");
      if (type) {
        inputBlock.push(`# Type: ${type}`);
      }
      inputBlock.push(`# Required: ${isRequired}`);
      if (defaultVal) {
        inputBlock.push(`# Default: ${defaultVal}`);
      }

      codeBlockArray.push(...inputBlock.map((l) => `${indent}${l}`));
      codeBlockArray.push("");
    }
    if (inputs.rows.length > 0) {
      codeBlockArray = codeBlockArray.slice(0, -1);
    }
  }

  codeBlockArray.push("```");

  // Create final resulting code block
  let result = "";
  for (const line of codeBlockArray) {
    result = `${result}${line}${getLineBreak(options.lineBreaks)}`;
  }
  return result;
}

function getToc(tocLevel: number): string {
  let result = "";
  for (let i = 0; i < tocLevel; i++) {
    result = `${result}#`;
  }
  return result;
}

export async function generateActionMarkdownDocs(
  inputOptions?: Options,
): Promise<string> {
  const options: DefaultOptions = {
    ...defaultOptions,
    ...inputOptions,
  };

  const docs = generateDocs(options);
  let outputString = "";

  for (const key in docs) {
    const value = docs[key];

    if (options.updateReadme) {
      await updateReadme(options, value, key, options.sourceFile);
    }

    outputString += value;
  }

  if (options.updateReadme) {
    await updateReadme(options, outputString, "all", options.sourceFile);
  }

  return outputString;
}

function generateDocs(options: DefaultOptions): Record<string, string> {
  const yml = parse(readFileSync(options.sourceFile, "utf-8")) as YmlStructure;

  if (yml.runs === undefined) {
    return generateWorkflowDocs(yml, options);
  } else {
    return generateActionDocs(yml, options);
  }
}

function generateActionDocs(
  yml: YmlStructure,
  options: DefaultOptions,
): Record<string, string> {
  return {
    header: generateHeader(yml, options),
    description: createMarkdownSection(options, yml.description, "Description"),
    inputs: generateInputs(yml.inputs, options, InputOutputType.actionInput),
    outputs: generateOutputs(yml.outputs, options),
    runs: createMarkdownSection(
      options,
      // eslint-disable-next-line i18n-text/no-en
      `This action is a \`${yml.runs.using}\` action.`,
      "Runs",
    ),
    usage: generateUsage(yml.inputs, options),
  };
}

function generateWorkflowDocs(
  yml: YmlStructure,
  options: DefaultOptions,
): Record<string, string> {
  return {
    header: generateHeader(yml, options),
    inputs: generateInputs(
      yml.on.workflow_call?.inputs,
      options,
      InputOutputType.workflowInput,
    ),
    secrets: generateSecrets(yml.on.workflow_call?.secrets, options),
    outputs: generateOutputs(yml.on.workflow_call?.outputs, options),
    runs: "",
    usage: generateUsage(yml.on.workflow_call?.inputs, options, false),
  };
}

function generateHeader(yml: YmlStructure, options: DefaultOptions): string {
  let header = "";
  if (options.includeNameHeader) {
    header = createMarkdownHeader(options, yml.name);
    options.tocLevel++;
  }

  return header;
}

function generateInputs(
  data: ActionInputsOutputs,
  options: DefaultOptions,
  type: InputOutputType,
): string {
  const inputMdTable = createMdTable(data, options, type);
  return createMarkdownSection(options, inputMdTable, "Inputs");
}

function generateSecrets(
  data: ActionInputsOutputs,
  options: DefaultOptions,
): string {
  const secretMdTable = createMdTable(
    data,
    options,
    InputOutputType.workflowSecret,
  );
  return createMarkdownSection(options, secretMdTable, "Secrets");
}

function generateOutputs(
  data: ActionInputsOutputs,
  options: DefaultOptions,
): string {
  const outputMdTable = createMdTable(
    data,
    options,
    InputOutputType.actionOutput,
  );
  return createMarkdownSection(options, outputMdTable, "Outputs");
}

function generateUsage(
  data: ActionInputsOutputs,
  options: DefaultOptions,
  isAction = true,
): string {
  const usageMdCodeBlock = createMdCodeBlock(data, options, isAction);
  return createMarkdownSection(options, usageMdCodeBlock, "Usage");
}

function escapeRegExp(x: string): string {
  return x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); // $& means the whole matched string
}

async function updateReadme(
  options: DefaultOptions,
  text: string,
  section: string,
  sourceFile: string,
): Promise<void> {
  const lineBreak = getLineBreak(options.lineBreaks);

  const readmeFileText = String(readFileSync(options.readmeFile, "utf-8"));
  const sourceOrActionMatches = readmeFileText.match(
    new RegExp(`<!-- action-docs-${section} (source|action)`),
  ) as string[];

  if (sourceOrActionMatches) {
    const sourceOrAction = sourceOrActionMatches[1];
    const matchProjectVersion = readmeFileText.match(
      new RegExp(
        `<!-- action-docs-${section} ${sourceOrAction}="${escapeRegExp(sourceFile)}" project="(.*)" version="(.*)" -->.?`,
      ),
    ) as string[];

    let commentExpression = `<!-- action-docs-${section} ${sourceOrAction}="${sourceFile}" PROJECT_VERSION-->`;
    commentExpression = commentExpression.replace(
      "PROJECT_VERSION",
      matchProjectVersion
        ? `project="${matchProjectVersion[1]}" version="${matchProjectVersion[2]}" `
        : "",
    );

    const regexp = new RegExp(
      `${escapeRegExp(commentExpression)}(?:(?:\r\n|\r|\n.*)+${escapeRegExp(commentExpression)})?`,
    );

    const processedText = text
      .trim()
      .replace(
        "***PROJECT***",
        matchProjectVersion ? matchProjectVersion[1] : "",
      )
      .replace(
        "***VERSION***",
        matchProjectVersion ? matchProjectVersion[2] : "",
      );

    await replaceInFile.replaceInFile({
      files: options.readmeFile,
      from: regexp,
      to:
        commentExpression +
        lineBreak +
        processedText +
        lineBreak +
        commentExpression,
    });
  }
}

function createMarkdownSection(
  options: DefaultOptions,
  data: string,
  header: string,
): string {
  const lineBreak = getLineBreak(options.lineBreaks);

  return data === "" || data === undefined
    ? ""
    : `${createMarkdownHeader(options, header)}${data}` +
        `${lineBreak}` +
        `${lineBreak}`;
}

function createMarkdownHeader(options: DefaultOptions, header: string): string {
  const lineBreak = getLineBreak(options.lineBreaks);

  return `${getToc(options.tocLevel)} ${header}${lineBreak}${lineBreak}`;
}

function isHtmlColumn(columnName: string): boolean {
  return columnName === "description";
}

function stripNewLines(value: string): string {
  return value.replace(/\r\n|\r|\n/g, " ");
}

function getInputOutput(
  data: ActionInputsOutputs,
  type: InputOutputType,
  format = true,
): { headers: string[]; rows: string[][] } {
  let headers: string[] = [];
  const rows: string[][] = [];

  if (data === undefined) {
    return { headers, rows };
  }

  headers = inputOutputHeaders[type];

  for (let i = 0; i < Object.keys(data).length; i++) {
    const key = Object.keys(data)[i];
    const value = data[key] as Record<string, string>;
    rows[i] = [];

    for (const columnName of headers) {
      let rowValue = "";

      if (columnName === "name") {
        rowValue = key;
      } else if (columnName === "description") {
        rowValue = value[columnName];
        if (value["deprecationMessage"] !== undefined) {
          rowValue += "<br/>_Deprecated";
          if (value["deprecationMessage"] !== "") {
            rowValue += `: ${value["deprecationMessage"]}`;
          }
          rowValue += "_";
        }
      } else if (columnName === "default") {
        rowValue =
          value[columnName] !== undefined && value[columnName] !== ""
            ? stripNewLines(String(value[columnName]))
            : inputOutputDefaults[columnName];
      } else {
        rowValue = value[columnName]
          ? value[columnName]
          : inputOutputDefaults[columnName];
      }

      if (format) {
        if (isHtmlColumn(columnName)) {
          rowValue = stripNewLines(converter.makeHtml(rowValue)).trim();
        } else {
          rowValue = `\`${rowValue}\``;
        }
      }

      rows[i].push(rowValue);
    }
  }
  return { headers, rows };
}
