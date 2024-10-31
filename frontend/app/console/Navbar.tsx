import { DarkModeToggle } from "@/components/color-mode-toggle";
import { UserButton } from "@clerk/nextjs";

const Navbar = () => {
  return (
    <div className="w-full items-center justify-center flex px-5 py-2 border-b min-h-[53px]">

      <div className="flex w-full justify-between max-w-screen-2xl">
        <div className="flex flex-row items-center justify-center gap-2">
          <h1 className="text-xl font-[family-name:var(--font-geist-mono)]">
            guard
          </h1>
        </div>

        <div className="flex flex-row items-center justify-center gap-2">
          <DarkModeToggle />
          <UserButton />
        </div>
      </div>
    </div>
  )
};

export default Navbar;
