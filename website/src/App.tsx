import { SiteHeader } from "@/components/sections/SiteHeader";
import { Hero } from "@/components/sections/Hero";
import { ProductPreview } from "@/components/sections/ProductPreview";
import { Features } from "@/components/sections/Features";
import { Steps, Scenes } from "@/components/sections/Workflow";
import { StorageSupport } from "@/components/sections/StorageSupport";
import { TechStack } from "@/components/sections/TechStack";
import { DeploySection } from "@/components/sections/DeploySection";
import { MigrationHighlight } from "@/components/sections/MigrationHighlight";
import { CtaBand } from "@/components/sections/CtaBand";
import { SiteFooter } from "@/components/sections/SiteFooter";

export default function App() {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <SiteHeader />

      <main className="flex-1 pb-20">
        <Hero />
        <ProductPreview />
        <Features />
        <Steps />
        <Scenes />
        <StorageSupport />
        <TechStack />
        <DeploySection />
        <MigrationHighlight />
        <CtaBand />
      </main>

      <SiteFooter />
    </div>
  );
}
