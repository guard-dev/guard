"use client"

import * as React from "react"
import {
  CaretSortIcon,
} from "@radix-ui/react-icons"

import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table"

import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { SCAN_ITEMS } from "./page"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import { Input } from "@/components/ui/input"
import { Check, Loader2 } from "lucide-react"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import Link from "next/link"

const scanColumns: ColumnDef<SCAN_ITEMS>[] = [
  {
    accessorKey: "service",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Service
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    enableGlobalFilter: true, // Add this line
  },
  {
    accessorKey: "region",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Region
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    enableGlobalFilter: true, // Add this line
  },
  {
    accessorKey: "summary",
    header: "Summary",
    cell: ({ row }) => (
      <div className="max-w-[300px] truncate" title={row.getValue("summary")}>
        {row.getValue("summary")}
      </div>
    ),
  },
  {
    accessorKey: "findings",
    header: "Findings",
    cell: ({ row }) => (row.getValue("findings") as string[]).length,
  },
  {
    accessorKey: "resourceCost",
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        >
          Resource Cost
          <CaretSortIcon className="ml-2 h-4 w-4" />
        </Button>
      )
    },
    cell: ({ row }) => {
      const cost = row.getValue("resourceCost") as number;
      return cost ? Math.round(cost) : '-';
    },
  },
]

interface DataTableProps {
  data: SCAN_ITEMS[]
  scanCompleted: boolean;
}

export function DataTable({ data: _data, scanCompleted }: DataTableProps) {
  const [sorting, setSorting] = React.useState<SortingState>([])
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [globalFilter, setGlobalFilter] = React.useState('')

  const cleanData = (data: SCAN_ITEMS) => ({
    ...data,
    resourceCost: data.resourceCost,
    region: cleanString(data.region),
    remedy: cleanString(data.remedy),
    service: cleanString(data.service),
    summary: cleanString(data.summary),
    findings: data.findings.map((f: string) => cleanString(f)),
    scanItemEntries: data.scanItemEntries.map((entry: any) => ({
      ...entry,
      findings: entry.findings.map((f: string) => cleanString(f)),
      title: cleanString(entry.title),
      summary: cleanString(entry.summary),
      remedy: cleanString(entry.remedy),
      commands: entry.commands.map((c: string) => cleanString(c)).filter((c: string) => c.length > 0),
    })),
  });

  const data = _data.map(cleanData);

  const table = useReactTable({
    data,
    columns: scanColumns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: (row, columnId, filterValue) => {
      const value = row.getValue(columnId) as string
      return value.toLowerCase().includes(filterValue.toLowerCase())
    },
    state: {
      sorting,
      columnFilters,
      globalFilter,
    },
  })

  return (
    <div className="w-full">
      <div className="flex items-center justify-between py-4">
        <Input
          placeholder="Search services or regions..."
          value={globalFilter}
          onChange={(event) => setGlobalFilter(event.target.value)}
          className="max-w-sm"
        />
        {!scanCompleted && (
          <div className="flex items-center text-muted-foreground">
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Scan in progress...
          </div>
        )}
        {scanCompleted && (
          <div className="flex items-center text-muted-foreground">
            <Check className="mr-2 h-4 w-4" />
            Scan completed
          </div>
        )}
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <Sheet key={row.id}>
                  <SheetTrigger asChild>
                    <TableRow className="cursor-pointer hover:bg-muted/50">
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id}>
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  </SheetTrigger>
                  <SheetContent className="w-full md:w-[800px] flex flex-col gap-8 md:max-w-[800px] overflow-y-auto">
                    <SheetHeader>
                      <SheetTitle>
                        {row.getValue("service")}
                      </SheetTitle>
                      <SheetDescription>
                        {row.getValue("region")}
                      </SheetDescription>
                    </SheetHeader>

                    <div className="flex flex-col gap-8">
                      <div className="flex flex-col gap-2 overflow-y-auto">
                        <div className="overflow-y-auto rounded max-h-[200px]">
                          {row.getValue("summary")}
                        </div>
                      </div>

                      <Accordion type="single" collapsible>
                        {row.original.scanItemEntries.length > 0 && (
                          row.original.scanItemEntries.map((entry, idx) => (
                            <AccordionItem value={entry.title} key={idx}>
                              <AccordionTrigger className="font-bold">{entry.title}</AccordionTrigger>
                              <AccordionContent>
                                <div className="flex flex-col gap-8" key={idx}>
                                  <div className="flex flex-col gap-2" key={idx}>
                                    <div className="overflow-y-auto rounded max-h-[200px]">
                                      {entry.summary}
                                    </div>
                                  </div>
                                  <div className="flex flex-col gap-2" key={idx}>
                                    <div className="font-bold">
                                      Action Items
                                    </div>
                                    <div className="overflow-y-auto rounded max-h-[200px]">
                                      {entry.remedy}
                                    </div>

                                    {entry.commands.length ?
                                      <>
                                        <div className="p-4 bg-neutral-800 text-neutral-100 rounded-md font-mono flex flex-col gap-5">
                                          {entry.commands.map((cmd: any, idx: number) => (
                                            <div key={idx}>
                                              <code key={idx}>
                                                {cmd}
                                              </code>
                                            </div>
                                          ))}
                                        </div>
                                        <div>
                                          <div>
                                            {"Disclaimer: These commands are for guidance only. Please review and validate them to ensure they align with your environment's specific configurations. "}
                                            <Link href={"https://docs.aws.amazon.com/cli/latest/reference/#available-services"} target="_blank" className="text-blue-700 underline">
                                              Refer to AWS CLI Documentation
                                            </Link>
                                          </div>
                                        </div>
                                      </>
                                      :
                                      <div />
                                    }


                                  </div>
                                </div>
                              </AccordionContent>
                            </AccordionItem>
                          ))
                        )}
                      </Accordion>
                    </div>
                  </SheetContent>
                </Sheet>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={scanColumns.length} className="h-24 text-center">
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-end space-x-2 py-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
        >
          Next
        </Button>
      </div>
    </div>
  )
}

const cleanString = (string: string) => {
  // First, replace escaped quotes (\" or \') with unescaped quotes
  let cleanedString = string.replace(/\\"/g, '"').replace(/\\'/g, "'");

  // Now, remove the first and last quote if they match
  if ((cleanedString.startsWith('"') && cleanedString.endsWith('"')) ||
    (cleanedString.startsWith("'") && cleanedString.endsWith("'"))) {
    cleanedString = cleanedString.slice(1, -1);
  }

  return cleanedString;
};

