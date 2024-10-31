import { Check, ChevronsUpDown } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Dispatch, SetStateAction, useState } from "react"


export function ComboboxDemo({ availableOptions, value, setValue }: { availableOptions: { value: string; label: string; }[]; value: string[]; setValue: Dispatch<SetStateAction<string[]>>; }) {
  const [open, setOpen] = useState(false)

  const handleSetValue = (val: string) => {
    if (value.includes(val)) {
      value.splice(value.indexOf(val), 1);
      setValue(value.filter((item) => item !== val));
    } else {
      setValue(prevValue => [...prevValue, val]);
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild className="max-w-[500px]">
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
        >
          <div className="flex gap-2 justify-start max-w-[400px] overflow-x-hidden">
            {value?.length ?
              value.map((val, i) => (
                <div key={i} className="px-2 py-1 rounded-xl border text-xs font-medium">{availableOptions.find((opt) => opt.value === val)?.label}</div>
              ))
              : "Select..."}
          </div>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[430px] p-0">
        <Command>
          <CommandInput placeholder="Search..." />
          <CommandEmpty>No framework found.</CommandEmpty>
          <CommandList>
            <CommandGroup>
              {availableOptions.map((opt) => (
                <CommandItem
                  key={opt.value}
                  value={opt.value}
                  onSelect={(currVal) => {
                    handleSetValue(currVal);
                  }}>
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value.includes(opt.value) ? "opacity-100" : "opacity-0"
                    )} />
                  {opt.label}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
