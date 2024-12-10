export type LineBreakType = "CR" | "LF" | "CRLF";

export function getLineBreak(lineBreakType: LineBreakType): string {
  switch (lineBreakType) {
    case "CR":
      return "\r";
    case "LF":
      return "\n";
    case "CRLF":
      return "\r\n";
  }
}

export function getLineBreakType(value: string): LineBreakType {
  if (isLineBreakType(value)) {
    return value;
  } else {
    return "LF" as LineBreakType;
  }
}

function isLineBreakType(value: string): value is LineBreakType {
  return value === "CR" || value === "LF" || value === "CRLF";
}
