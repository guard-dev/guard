export const FooterSection = () => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-center font-[family-name:var(--font-geist-mono)] text-sm border-t border-neutral-200 px-5 py-5 w-full">
      <div className="mb-4 md:mb-0">
        <p>&copy; 2024 Guard. All rights reserved.</p>
      </div>
      <div className="flex space-x-6">
        <a href="https://discord.gg/XCKnF8k2hF" className="hover:underline">Community Discord</a>
        <a href="https://www.github.com/guard-dev/guard" className="hover:underline">GitHub</a>
        <a href="https://www.cal.com/guard" className="hover:underline">Contact Us</a>
      </div>
    </div>
  );
};
