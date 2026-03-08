export type Column = {
  key: string;
  label: string;
  width?: string;
  headerClass?: string;
  rowClass?: string;
  class?: string;
  sortable?: boolean;
  isSearchable?: boolean;
  type?: "boolean" | "string" | "number" | "date";
  show?: boolean; // Show/hide column (default: true)
};

export type TableRow = Record<string, unknown>;
