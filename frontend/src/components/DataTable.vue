<template>
  <div class="table-responsive table-container">
    <table class="w-100 table table-hover table-rounded">
      <thead>
        <!-- Search Inputs/Checkboxes Row -->
        <tr class="align-middle">
          <th class="user-select-none" v-for="column in visibleColumns" :key="column.key" :rowspan="column.isSearchable ? 1 : 2" :style="{ width: column.width }" :class="[column.headerClass, column.isSearchable ? 'searchable-column' : '', column.sortable ? 'cursor-pointer' : 'user-select-none']">
            <!-- Search input for searchable columns -->
            <div v-if="column.isSearchable" class="search-container">
              <!-- Checkbox for boolean columns -->
              <input v-if="column.type === 'boolean'" type="checkbox" v-model="searchQueries[column.key]" @click.stop="handleCheckboxChange(column.key)" />
              <!-- Input for non-boolean columns -->
              <input v-if="column.type !== 'boolean'" v-model="searchQueries[column.key]" type="text" :placeholder="column.label" class="form-control border-primary-subtle" />
            </div>
            <div v-else :class="{ 'cursor-pointer': column.sortable }" @click="column.sortable ? toggleSort(column.key) : undefined">
              <slot :name="`header-${column.key}`">
                {{ column.label }}
              </slot>
              <span v-if="sortKey === column.key">
                <span v-if="sortOrder === 'asc'">↑</span>
                <span v-if="sortOrder === 'desc'">↓</span>
              </span>
            </div>
          </th>
        </tr>

        <!-- Column Titles Row -->
        <tr>
          <template v-for="column in visibleColumns" :key="`header-${column.key}`">
            <th class="user-select-none" v-if="column.isSearchable" :style="{ width: column.width }" :class="[column.headerClass, column.sortable ? 'cursor-pointer' : 'user-select-none', sortKey === column.key && sortedClass]" @click="column.sortable ? toggleSort(column.key) : undefined" :rowspan="column.sortable ? 1 : 2">
              <!-- Set rowspan for sortable columns -->
              <!-- Column title with sorting click event -->
              <slot :name="`header-${column.key}`">
                {{ column.label }}
              </slot>
              <span v-if="sortKey === column.key">
                <span v-if="sortOrder === 'asc'">↑</span>
                <span v-if="sortOrder === 'desc'">↓</span>
              </span>
            </th>
          </template>
        </tr>
      </thead>
      <tbody>
        <tr class="align-middle" v-if="currentPageRows.length === 0">
          <td :colspan="visibleColumns.length">Empty</td>
        </tr>
        <tr class="align-middle" v-else v-for="(row, rowIndex) in currentPageRows" :key="rowIndex">
          <td v-for="column in visibleColumns" :key="column.key" :class="[column.rowClass, sortKey === column.key && sortedClass]">
            <slot :name="`cell-${column.key}`" :value="row[column.key]" :row="row">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Pagination Controls -->
    <nav aria-label="Table Pagination">
      <ul class="pagination justify-content-center">
        <li class="page-item" :class="{ disabled: currentPage === 1 }">
          <button class="page-link" @click="changePage(currentPage - 1)" aria-label="Previous">
            <span aria-hidden="true">&laquo;</span>
          </button>
        </li>
        <template v-for="page in visiblePages" :key="page">
          <li v-if="page === '...'" class="page-item disabled">
            <span class="page-link">...</span>
          </li>
          <li v-else class="page-item" :class="{ active: page === currentPage }">
            <button class="page-link" @click="changePage(page as number)">{{ page }}</button>
          </li>
        </template>
        <li class="page-item" :class="{ disabled: currentPage === totalPages }">
          <button class="page-link" @click="changePage(currentPage + 1)" aria-label="Next">
            <span aria-hidden="true">&raquo;</span>
          </button>
        </li>
      </ul>

      <!-- Page Size Dropdown -->
      <div class="d-flex justify-content-center mt-3">
        <select class="form-select w-auto" v-model="pageSize" @change="onPageSizeChange">
          <option v-for="size in pageSizes" :key="size" :value="size">{{ size }} rows per page</option>
          <option value="-1">All</option>
          <!-- Add 'All' option -->
        </select>
      </div>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import type { Column, TableRow } from "../types/table";

type SortOrder = "asc" | "desc" | null;

const sortedClass = "col-sorted";

const props = defineProps<{
  columns: Column[];
  data: TableRow[];
  defaultSortKey?: string;
  defaultSortOrder?: SortOrder;
  pageSize?: number; // Make pageSize optional
  storageKey?: string; // Optional key for localStorage
}>();

// Computed property for visible columns (show !== false defaults to true)
const visibleColumns = computed(() => props.columns.filter((col) => col.show !== false));

const emit = defineEmits<{
  (event: "page-change", page: number): void;
  (event: "page-size-change", size: number): void;
}>();

const STORAGE_KEY = props.storageKey || "datatable-sort";

// Load from localStorage or use defaults
const loadSortState = () => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const { sortKey: storedKey, sortOrder: storedOrder } = JSON.parse(stored);
      // Validate that the stored sortKey exists in current columns
      const isValidKey = storedKey && props.columns.some((col) => col.key === storedKey);
      return {
        sortKey: isValidKey ? storedKey : (props.defaultSortKey ?? null),
        sortOrder: isValidKey ? storedOrder : (props.defaultSortOrder ?? null)};
    }
  } catch (error) {
    console.error("Error loading sort state from localStorage:", error);
  }
  return {
    sortKey: props.defaultSortKey ?? null,
    sortOrder: props.defaultSortOrder ?? null};
};

const initialState = loadSortState();
const sortKey = ref<string | null>(initialState.sortKey);
const sortOrder = ref<SortOrder>(initialState.sortOrder);
const searchQueries = ref<Record<string, unknown>>({});
const currentPage = ref(1); // Track current page
const pageSize = ref<number>(props.pageSize ?? 10); // Default page size to 10 if not provided
const pageSizeToNumber = computed(() => Number(pageSize.value));

const pageSizes = [10, 20, 50, 100]; // Options for page size

// Save to localStorage whenever sort state changes
watch([sortKey, sortOrder], () => {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        sortKey: sortKey.value,
        sortOrder: sortOrder.value}),
    );
  } catch (error) {
    console.error("Error saving sort state to localStorage:", error);
  }
});

onMounted(() => {
  if (sortKey.value && !sortOrder.value) {
    sortOrder.value = "asc";
  }
});

const compareValues = (a: unknown, b: unknown, columnKey: string): number => {
  // Get column type from columns
  const column = props.columns.find((col) => col.key === columnKey);
  const columnType = column?.type;

  // Only convert dates if column type is explicitly 'date'
  if (columnType === "date") {
    if (typeof a === "string") a = new Date(a);
    if (typeof b === "string") b = new Date(b);
  }

  // Date comparison
  if (a instanceof Date && b instanceof Date) {
    return sortOrder.value === "asc" ? a.getTime() - b.getTime() : b.getTime() - a.getTime();
  }

  // Only convert to numbers if column type is explicitly 'number'
  if (columnType === "number") {
    if (typeof a === "string") a = parseFloat(a);
    if (typeof b === "string") b = parseFloat(b);
  }

  // Number comparison
  if (typeof a === "number" && typeof b === "number") {
    return sortOrder.value === "asc" ? a - b : b - a;
  }

  // Boolean comparison
  if (typeof a === "boolean" && typeof b === "boolean") {
    return sortOrder.value === "asc" ? Number(a) - Number(b) : Number(b) - Number(a);
  }

  // Default case (string sorting)
  if (typeof a === "string" && typeof b === "string") {
    return sortOrder.value === "asc" ? a.localeCompare(b) : b.localeCompare(a);
  }

  return 0;
};

const sortedData = computed(() => {
  if (!sortKey.value || !sortOrder.value) {
    return props.data;
  }

  return [...props.data].sort((a, b) => {
    const valueA = a[sortKey.value!] as unknown;
    const valueB = b[sortKey.value!] as unknown;

    return compareValues(valueA, valueB, sortKey.value!);
  });
});

const filteredData = computed(() => {
  const result = sortedData.value.filter((row) => {
    return props.columns.every((column) => {
      const searchQuery = searchQueries.value[column.key];
      const cellValue = row[column.key];

      if (!searchQuery && searchQuery !== false && searchQuery !== true) {
        return true;
      }

      if (cellValue instanceof Date) {
        return cellValue.toLocaleDateString("en-GB").includes(searchQuery as string);
      }

      if (typeof searchQuery === "boolean" && typeof cellValue === "boolean") {
        return cellValue === searchQuery;
      }

      return containsAllSubstrings(String(cellValue).trim().toLowerCase(), toSearchTermsArray(searchQuery as string));
    });
  });

  // Reset to page 1 if filter changed and current page is out of bounds
  const maxPage = Math.ceil(result.length / pageSize.value) || 1;
  if (currentPage.value > maxPage) {
    currentPage.value = 1;
  }

  return result;
});

const toSearchTermsArray = (str: string): string[] => str.trim().toLowerCase().split(/\s+/);

/**
 * Checks if a string A contains a list of all substrings.
 * @param str
 * @param substrings
 */
const containsAllSubstrings = (str: string, substrings: string[]): boolean => substrings.every((sub) => str.indexOf(sub) !== -1);

const totalPages = computed(() => {
  if (pageSizeToNumber.value === -1) {
    return 1; // All items fit in one page
  }
  return Math.ceil(filteredData.value.length / pageSize.value);
});

const visiblePages = computed(() => {
  const total = totalPages.value;
  const current = currentPage.value;
  const maxVisible = 7; // Maximum number of page buttons to show

  if (total <= maxVisible) {
    // Show all pages if total is less than maxVisible
    return Array.from({ length: total }, (_, i) => i + 1);
  }

  const pages: (number | string)[] = [];

  // Always show first page
  pages.push(1);

  if (current > 3) {
    pages.push("...");
  }

  // Show pages around current page
  const start = Math.max(2, current - 1);
  const end = Math.min(total - 1, current + 1);

  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  if (current < total - 2) {
    pages.push("...");
  }

  // Always show last page
  if (total > 1) {
    pages.push(total);
  }

  return pages;
});

const currentPageRows = computed(() => {
  if (pageSizeToNumber.value === -1) {
    return filteredData.value; // Show all rows
  }
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredData.value.slice(start, end);
});

const toggleSort = (key: string) => {
  if (sortKey.value === key) {
    sortOrder.value = sortOrder.value === "asc" ? "desc" : sortOrder.value === "desc" ? null : "asc";
    if (!sortOrder.value) {
      sortKey.value = null;
    }
  } else {
    sortKey.value = key;
    sortOrder.value = "asc";
  }
};

const handleCheckboxChange = (key: string) => {
  searchQueries.value[key] = !searchQueries.value[key];
};

const changePage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
    emit("page-change", page); // Emit page change event
  }
};

const onPageSizeChange = () => {
  if (pageSizeToNumber.value !== -1) {
    emit("page-size-change", pageSize.value); // Emit page size change event
    currentPage.value = 1; // Reset to page 1 when page size changes
  } else {
    emit("page-size-change", -1); // Emit "all" for page size
  }
};
</script>

<style scoped lang="scss">
@use "@/assets/custom-bootstrap.scss" as bootstrap;

[data-bs-theme="light"] {
  table .col-sorted {
    background-color: bootstrap.$light;
  }

  table th {
    background-color: bootstrap.$light;
  }

  table {
    border: 1px solid bootstrap.$secondary;
  }

  table tbody td {
    border-top: 1px solid bootstrap.$secondary;
  }

  table th,
  table td {
    border-right: 1px solid bootstrap.$secondary;
  }
}

[data-bs-theme="dark"] {
  table tbody td {
    border-top: 1px solid bootstrap.$info;
  }

  .table .col-sorted {
    background-color: bootstrap.$primary;
  }

  table th {
    background-color: bootstrap.$primary !important;
    border: none;
  }

  table th input {
    border: 1px solid bootstrap.$info !important;
  }

  table {
    border: 1px solid bootstrap.$info;
  }

  table th + th,
  table td + td {
    border-left: 1px solid bootstrap.$info;
  }
}

.cursor-pointer {
  cursor: pointer;
}

.table-rounded {
  border-radius: 5px;
  border-collapse: separate;
  border-spacing: 0;
  overflow: hidden;
}

.table-rounded thead tr:first-child th:first-child {
  border-top-left-radius: 5px;
}

.table-rounded thead tr:first-child th:last-child {
  border-top-right-radius: 5px;
}

.table-rounded tbody tr:last-child td:first-child {
  border-bottom-left-radius: 5px;
}

.table-rounded tbody tr:last-child td:last-child {
  border-bottom-right-radius: 5px;
}

.search-container input {
  width: 100%;
  padding: 5px;
}

.search-container input[type="checkbox"] {
  width: auto;
}

/* Optional: Style for column headers with sorting functionality */
.column-header-label {
  cursor: pointer;
  display: inline-block;
}

.searchable-column {
  width: 100px;
}
</style>

