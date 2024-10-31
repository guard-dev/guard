import { Button } from "@/components/ui/button";
import PricingTable from "@/components/PricingTable";
import { IconCloudSecurity, IconAI, IconRealTime, IconActionable } from '@/components/Icons';
import Navbar from "./console/Navbar";

export default function Home() {
  return (
    <div className="flex items-center justify-center pt-[200px] md:min-h-screen md:pt-0 font-[family-name:var(--font-geist-sans)] p-5">
      <main className="flex flex-col gap-8 row-start-2 items-center sm:items-start">
        <Navbar />
        <div className="pt-[20px] md:pt-[120px]">
          <HeroSection />
        </div>

        <div className="pt-[120px] max-w-[982px] w-full">
          <FeaturesSection />
        </div>

        <div className="flex w-full h-full pt-[120px] max-w-[982px]">
          <PricingTable />
        </div>

        <footer className="w-full max-w-[982px] py-8 mt-16 pt-[120px]">
          <FooterSection />
        </footer>
      </main>
    </div>
  );
}

const HeroSection = () => {
  return (
    <div className="flex flex-col w-full h-full gap-8">
      <h1 className="text-5xl font-[family-name:var(--font-geist-mono)] w-full">
        guard
      </h1>
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-[family-name:var(--font-geist-mono)]">
          AI-Powered Cloud Security, Simplified.
        </h1>
        <h2 className="text-lg font-[family-name:var(--font-geist-mono)]">
          Detect AWS misconfigurations in real-time and fix them with AI-driven, actionable insights.
        </h2>
      </div>

      <div className="flex gap-4 items-center flex-row w-full">
        <a
          href="/console"
        >
          <Button className="rounded-full">
            Get Started
          </Button>
        </a>
        <a
          href="https://www.github.com/guard-dev/guard"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button className="rounded-full" variant={"outline"}>
            Self Host
          </Button>
        </a>
      </div>
    </div>
  );
}

const FeaturesSection = () => {
  const features = [
    {
      icon: <IconCloudSecurity className="w-12 h-12 mb-4" />,
      title: "Cloud Security Scanning",
      description: "Comprehensive scanning of your AWS resources to identify potential security risks and misconfigurations."
    },
    {
      icon: <IconAI className="w-12 h-12 mb-4" />,
      title: "AI-Powered Analysis",
      description: "Leverage advanced LLMs to process and analyze your cloud infrastructure for deeper insights."
    },
    {
      icon: <IconRealTime className="w-12 h-12 mb-4" />,
      title: "Real-Time Detection",
      description: "Instantly identify vulnerabilities and security issues as they arise in your AWS environment."
    },
    {
      icon: <IconActionable className="w-12 h-12 mb-4" />,
      title: "Actionable Insights",
      description: "Receive clear, actionable recommendations to improve your cloud security posture."
    }
  ];

  return (
    <div className="w-full max-w-5xl mx-auto font-[family-name:var(--font-geist-mono)]">
      <h2 className="text-3xl font-medium text-left mb-12">Key Features</h2>
      <div className="flex flex-col gap-8">
        {features.map((feature, index) => (
          <div key={index} className="flex flex-row items-start gap-6">
            <div className="flex-shrink-0">
              {feature.icon}
            </div>
            <div className="text-left">
              <h3 className="text-xl font-bold mb-2">{feature.title}</h3>
              <p className="text-sm">{feature.description}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const FooterSection = () => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-center font-[family-name:var(--font-geist-mono)] text-sm border-t border-neutral-200 pt-[40px]">
      <div className="mb-4 md:mb-0">
        <p>&copy; 2024 Guard. All rights reserved.</p>
      </div>
      <div className="flex space-x-6">
        <a href="/privacy" className="hover:underline">Privacy Policy</a>
        <a href="/terms" className="hover:underline">Terms of Service</a>
        <a href="/contact" className="hover:underline">Contact Us</a>
      </div>
    </div>
  );
};
