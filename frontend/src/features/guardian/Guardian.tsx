import { useMemo, useState } from "react";
import { useCharacters } from "../../contexts/CharacterContext";
import { useGuardianEquipment } from "../../data/characters";
import type { EquipmentGroup, EquippedItem } from "../../types/design";
import { Icon } from "../../components/Icon";
import { QueryErrorPanel } from "../../components/QueryErrorPanel";
import { EmptyState, ItemTile, Skeleton } from "../../components/primitives";

const GROUPS: EquipmentGroup[] = ["Weapons", "Armor", "Equipment"];

function GuardianHeroArt({ src }: { src: string | undefined }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) return null;
  return (
    <img
      className="gt-guardian-hero-art"
      src={src}
      alt=""
      onError={() => setFailed(true)}
    />
  );
}

function GuardianMark({ src }: { src: string | undefined }) {
  const [failed, setFailed] = useState(false);
  return (
    <div className="gt-guardian-mark" aria-hidden="true">
      {src && !failed ? (
        <img src={src} alt="" onError={() => setFailed(true)} />
      ) : (
        <Icon name="guardian" size="2rem" stroke={1.5} />
      )}
    </div>
  );
}

function EquipmentRow({ item }: { item: EquippedItem }) {
  return (
    <article className="gt-guardian-item" data-rarity={item.rarity}>
      <ItemTile
        rarity={item.rarity}
        type={item.type}
        icon={item.icon}
        style={{ width: "3rem" }}
      />
      <div className="gt-guardian-item-main">
        <span className="gt-guardian-slot">{item.slot}</span>
        <h3 className="gt-guardian-item-name">{item.name}</h3>
        <span className="gt-guardian-item-type">{item.type}</span>
      </div>
      {item.power != null && (
        <span
          className="gt-guardian-item-power mono"
          aria-label={`${item.power} Power`}
        >
          <Icon name="bolt" size="0.75rem" />
          {item.power}
        </span>
      )}
    </article>
  );
}

function EquipmentLoading() {
  return (
    <div className="gt-guardian-groups" aria-label="Loading equipment">
      {GROUPS.map((group) => (
        <section className="gt-guardian-group" key={group}>
          <Skeleton w="6rem" h="1rem" />
          <div className="gt-guardian-list">
            {[0, 1, 2].map((row) => (
              <div className="gt-guardian-item" key={row}>
                <Skeleton w="3rem" h="3rem" r="var(--r-sm)" />
                <div className="gt-guardian-item-main">
                  <Skeleton w="3rem" h="0.6rem" />
                  <Skeleton w="70%" h="0.9rem" />
                  <Skeleton w="45%" h="0.7rem" />
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

export function Guardian() {
  const { activeCharacter, isLoading: rosterLoading } = useCharacters();
  const {
    equipment,
    isLoading: equipmentLoading,
    isError,
    error,
    retry,
  } = useGuardianEquipment(activeCharacter?.id);

  const grouped = useMemo(() => {
    const result = new Map<EquipmentGroup, EquippedItem[]>();
    for (const group of GROUPS) result.set(group, []);
    for (const item of equipment?.items ?? []) {
      result.get(item.group)?.push(item);
    }
    return result;
  }, [equipment]);

  if (rosterLoading) {
    return (
      <div className="gt-page">
        <Skeleton w="100%" h="10rem" r="var(--r-lg)" />
        <EquipmentLoading />
      </div>
    );
  }

  if (!activeCharacter) {
    return (
      <div className="gt-page">
        <div className="gt-card">
          <EmptyState
            icon="guardian"
            title="No Guardians found"
            body="Bungie did not return a playable character for this Destiny membership."
          />
        </div>
      </div>
    );
  }

  return (
    <div className="gt-page">
      <header className="gt-guardian-hero">
        <GuardianHeroArt
          key={activeCharacter.emblemBackgroundUrl ?? activeCharacter.id}
          src={activeCharacter.emblemBackgroundUrl}
        />
        <div className="gt-guardian-hero-content">
          <GuardianMark
            key={activeCharacter.emblemUrl ?? activeCharacter.id}
            src={activeCharacter.emblemUrl}
          />
          <div className="gt-guardian-identity">
            <h1>{activeCharacter.cls}</h1>
            <p>{activeCharacter.race} Guardian</p>
          </div>
          <div className="gt-guardian-power">
            <Icon name="bolt" size="1rem" />
            <strong className="mono">{activeCharacter.power}</strong>
            <span>Power</span>
          </div>
        </div>
        <p className="gt-guardian-scope">
          Equipment follows the selected Guardian. Collections remain shared
          across your Destiny membership.
        </p>
      </header>

      {isError ? (
        <QueryErrorPanel error={error} onRetry={retry} />
      ) : equipmentLoading || !equipment ? (
        <EquipmentLoading />
      ) : equipment.state === "unavailable" ? (
        <div className="gt-card">
          <EmptyState
            icon="info"
            title="Equipment is unavailable"
            body="Bungie did not return equipment for this Guardian. Try again after reconnecting or refreshing your Destiny data."
          />
        </div>
      ) : equipment.items.length === 0 ? (
        <div className="gt-card">
          <EmptyState
            icon="guardian"
            title="No equipped items returned"
            body="Choose another Guardian or refresh your Destiny data."
          />
        </div>
      ) : (
        <div className="gt-guardian-groups">
          {GROUPS.map((group) => {
            const items = grouped.get(group) ?? [];
            if (items.length === 0) return null;
            return (
              <section className="gt-guardian-group" key={group}>
                <h2>{group}</h2>
                <div className="gt-guardian-list">
                  {items.map((item) => (
                    <EquipmentRow item={item} key={item.id} />
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      )}
    </div>
  );
}
