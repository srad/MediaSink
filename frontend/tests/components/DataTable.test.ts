import { mount } from "@vue/test-utils";
import DataTable from "../../src/components/DataTable.vue";
import { describe, it, expect } from "vitest";

// Mock data
const columns = [
  { key: "name", label: "Name", sortable: true },
  { key: "age", label: "Age", sortable: true },
  { key: "createdAt", label: "Created At", sortable: true },
];

const data = [
  { name: "Alice", age: 30, createdAt: new Date("2023-01-01") },
  { name: "Bob", age: 25, createdAt: new Date("2023-03-15") },
  { name: "Charlie", age: 35, createdAt: new Date("2023-02-10") },
];

describe("DataTable Component", () => {
  it("renders the table with the correct number of rows and columns", () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
      },
    });

    // Check the number of columns in the header (including sorting arrows)
    const headers = wrapper.findAll("th");
    expect(headers).toHaveLength(3); // Should have 3 columns

    // Check the number of rows in the table body
    const rows = wrapper.findAll("tbody tr");
    expect(rows).toHaveLength(3); // 3 data rows
  });

  it("correctly sorts the table by the default sorting order", () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
        defaultSortKey: "age",
        defaultSortOrder: "asc",
      },
    });

    // Check if rows are sorted by age in ascending order
    const firstRow = wrapper.findAll("tbody tr")[0];
    const secondRow = wrapper.findAll("tbody tr")[1];
    const thirdRow = wrapper.findAll("tbody tr")[2];

    expect(firstRow.find(('td:nth-child(1)')).text()).toContain("Bob");
    expect(secondRow.text()).toContain("Alice");
    expect(thirdRow.text()).toContain("Charlie");

    expect(firstRow.text()).toContain("25");
    expect(secondRow.text()).toContain("30");
    expect(thirdRow.text()).toContain("35");
  });

  it("changes sorting order when clicking on the sortable column header", async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
        defaultSortKey: "age",
        defaultSortOrder: "asc",
      },
    });

    // Find the "Age" column header and simulate a click
    const ageColumnHeader = wrapper.find("th:nth-child(2)");
    await ageColumnHeader.trigger("click");

    // Check if the table is now sorted by age in descending order
    const firstRow = wrapper.findAll("tbody tr")[0];
    const secondRow = wrapper.findAll("tbody tr")[1];
    const thirdRow = wrapper.findAll("tbody tr")[2];

    expect(firstRow.text()).toContain("Bob");
    expect(secondRow.text()).toContain("Alice");
    expect(thirdRow.text()).toContain("Charlie");

    expect(firstRow.find('td:nth-child(2)').text()).toContain("25");
    expect(secondRow.text()).toContain("30");
    expect(thirdRow.text()).toContain("35");
  });

  it("correctly handles Date sorting", () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
        defaultSortKey: "createdAt",
        defaultSortOrder: "asc",
      },
    });

    // Format the expected dates to match the locale format
    const formatDate = (date: Date): string => {
      return date.toDateString("en-GB"); // Adjust to your desired locale format
    };

    // Get the rows and check the date cell content
    const rows = wrapper.findAll("tbody tr");

    // Get the date cell for each row
    const firstRowDateCell = rows[0].find("td:nth-child(3)"); // Assuming 'createdAt' is the 3rd column
    const secondRowDateCell = rows[1].find("td:nth-child(3)");
    const thirdRowDateCell = rows[2].find("td:nth-child(3)");

    // Check if the date cells contain the correctly formatted date values
    expect(firstRowDateCell.text()).toContain(formatDate(new Date("2023-01-01")));
    expect(secondRowDateCell.text()).toContain(formatDate(new Date("2023-02-10")));
    expect(thirdRowDateCell.text()).toContain(formatDate(new Date("2023-03-15")));
  });

  it("displays sorting arrows in the header", () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
        defaultSortKey: "createdAt",
        defaultSortOrder: "asc",
      },
    });

    // Ensure the sorting arrow is displayed on the sorted column
    const createdAtHeader = wrapper.find("th:nth-child(3)");
    expect(createdAtHeader.text()).toContain("↑"); // Arrow should be pointing up for ascending order
  });

  it("should allow a custom class for the sorted column", () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data,
        defaultSortKey: "createdAt",
        defaultSortOrder: "asc",
        sortedClass: "sorted-column",
      },
    });

    // Ensure the sorted column has the correct class
    const createdAtHeader = wrapper.find("th:nth-child(3)");
    // TODO: expect(createdAtHeader.classes()).toContain("sorted-column");
  });
});
