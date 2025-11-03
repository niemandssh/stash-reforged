import React, {
  useEffect,
  useMemo,
  useState,
  useCallback,
  useRef,
} from "react";
import { Tabs, Tab, Col, Row } from "react-bootstrap";
import { useIntl } from "react-intl";
import { useHistory, Redirect, RouteComponentProps } from "react-router-dom";
import { Helmet } from "react-helmet";
import cx from "classnames";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import {
  useFindPerformer,
  usePerformerUpdate,
  usePerformerDestroy,
  mutateMetadataAutoTag,
  usePerformerProfileImageUpdate,
  usePerformerProfileImageDestroy,
} from "src/core/StashService";
import { DetailsEditNavbar } from "src/components/Shared/DetailsEditNavbar";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { ConfigurationContext } from "src/hooks/Config";
import { RatingSystem } from "src/components/Shared/Rating/RatingSystem";
import {
  CompressedPerformerDetailsPanel,
  PerformerDetailsPanel,
} from "./PerformerDetailsPanel";
import { PerformerScenesPanel } from "./PerformerScenesPanel";
import { PerformerGalleriesPanel } from "./PerformerGalleriesPanel";
import { PerformerGroupsPanel } from "./PerformerGroupsPanel";
import { PerformerImagesPanel } from "./PerformerImagesPanel";
import { PerformerAppearsWithPanel } from "./performerAppearsWithPanel";
import { PerformerEditPanel } from "./PerformerEditPanel";
import { PerformerSubmitButton } from "./PerformerSubmitButton";
import { useRatingKeybinds } from "src/hooks/keybinds";
import { ProfileImageSlider } from "./ProfileImageSlider";
import { useLoadStickyHeader } from "src/hooks/detailsPanel";
import { useScrollToTopOnMount } from "src/hooks/scrollToTop";
import { ExternalLinkButtons } from "src/components/Shared/ExternalLinksButton";
import { BackgroundImage } from "src/components/Shared/DetailsPage/BackgroundImage";
import {
  TabTitleCounter,
  useTabKey,
} from "src/components/Shared/DetailsPage/Tabs";
import { DetailTitle } from "src/components/Shared/DetailsPage/DetailTitle";
import { ExpandCollapseButton } from "src/components/Shared/CollapseButton";
import { FavoriteIcon } from "src/components/Shared/FavoriteIcon";
import { TagLink } from "src/components/Shared/TagLink";
import { AliasList } from "src/components/Shared/DetailsPage/AliasList";
import { HeaderImage } from "src/components/Shared/DetailsPage/HeaderImage";
import { PatchComponent } from "src/patch";
import { ILightboxImage } from "src/hooks/Lightbox/types";

interface IProps {
  performer: GQL.PerformerDataFragment;
  tabKey?: TabKey;
}

interface IPerformerParams {
  id: string;
  tab?: string;
}

const validTabs = [
  "default",
  "scenes",
  "galleries",
  "images",
  "groups",
  "appearswith",
] as const;
type TabKey = (typeof validTabs)[number];

function isTabKey(tab: string): tab is TabKey {
  return validTabs.includes(tab as TabKey);
}

const PerformerTabs: React.FC<{
  tabKey?: TabKey;
  performer: GQL.PerformerDataFragment;
  abbreviateCounter: boolean;
}> = ({ tabKey, performer, abbreviateCounter }) => {
  const populatedDefaultTab = useMemo(() => {
    let ret: TabKey = "scenes";
    if (performer.scene_count == 0) {
      if (performer.gallery_count != 0) {
        ret = "galleries";
      } else if (performer.image_count != 0) {
        ret = "images";
      } else if (performer.group_count != 0) {
        ret = "groups";
      }
    }

    return ret;
  }, [performer]);

  const { setTabKey } = useTabKey({
    tabKey,
    validTabs,
    defaultTabKey: populatedDefaultTab,
    baseURL: `/performers/${performer.id}`,
  });

  useEffect(() => {
    Mousetrap.bind("c", () => setTabKey("scenes"));
    Mousetrap.bind("g", () => setTabKey("galleries"));
    Mousetrap.bind("m", () => setTabKey("groups"));

    return () => {
      Mousetrap.unbind("c");
      Mousetrap.unbind("g");
      Mousetrap.unbind("m");
    };
  });

  return (
    <Tabs
      id="performer-tabs"
      mountOnEnter
      unmountOnExit
      activeKey={tabKey}
      onSelect={setTabKey}
    >
      <Tab
        eventKey="scenes"
        title={
          <TabTitleCounter
            messageID="scenes"
            count={performer.scene_count}
            abbreviateCounter={abbreviateCounter}
          />
        }
      >
        <PerformerScenesPanel
          active={tabKey === "scenes"}
          performer={performer}
        />
      </Tab>

      <Tab
        eventKey="galleries"
        title={
          <TabTitleCounter
            messageID="galleries"
            count={performer.gallery_count}
            abbreviateCounter={abbreviateCounter}
          />
        }
      >
        <PerformerGalleriesPanel
          active={tabKey === "galleries"}
          performer={performer}
        />
      </Tab>

      <Tab
        eventKey="images"
        title={
          <TabTitleCounter
            messageID="images"
            count={performer.image_count}
            abbreviateCounter={abbreviateCounter}
          />
        }
      >
        <PerformerImagesPanel
          active={tabKey === "images"}
          performer={performer}
        />
      </Tab>

      <Tab
        eventKey="groups"
        title={
          <TabTitleCounter
            messageID="groups"
            count={performer.group_count}
            abbreviateCounter={abbreviateCounter}
          />
        }
      >
        <PerformerGroupsPanel
          active={tabKey === "groups"}
          performer={performer}
        />
      </Tab>

      <Tab
        eventKey="appearswith"
        title={
          <TabTitleCounter
            messageID="appears_with"
            count={performer.performer_count}
            abbreviateCounter={abbreviateCounter}
          />
        }
      >
        <PerformerAppearsWithPanel
          active={tabKey === "appearswith"}
          performer={performer}
        />
      </Tab>
    </Tabs>
  );
};

interface IPerformerHeaderImageProps {
  activeImage: string | null | undefined;
  collapsed: boolean;
  encodingImage: boolean;
  lightboxImages: ILightboxImage[];
  performer: GQL.PerformerDataFragment;
  isEditing: boolean;
  isNew?: boolean;
  currentImageIndex?: number;
  onImageChange?: (index: number) => void;
  setImage?: (image?: string | null) => void;
  setEncodingImage?: (loading: boolean) => void;
  onPerformerUpdate?: (
    updatedPerformer: Partial<GQL.PerformerDataFragment>
  ) => void;
  onImageUpdate?: () => Promise<void>;
  onAddImage?: () => void;
  onImageFileChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
}

const PerformerHeaderImage: React.FC<IPerformerHeaderImageProps> =
  PatchComponent(
    "PerformerHeaderImage",
    ({
      encodingImage,
      activeImage,
      performer,
      isEditing,
      isNew,
      currentImageIndex,
      onImageChange,
      setImage,
      setEncodingImage,
      onPerformerUpdate,
      onImageUpdate: onImageUpdateProp,
      onAddImage,
    }) => {
      const intl = useIntl();
      const Toast = useToast();
      const hasProfileImages =
        performer.profile_images && performer.profile_images.length > 0;
      const [updateProfileImage] = usePerformerProfileImageUpdate();
      const [destroyProfileImage] = usePerformerProfileImageDestroy();

      const handleSetPrimary = useCallback(
        async (imageId: string) => {
          try {
            // First, set all other images to not primary
            const otherImages =
              performer.profile_images?.filter((img) => img.id !== imageId) ||
              [];
            for (const img of otherImages) {
              if (img.is_primary) {
                await updateProfileImage({
                  variables: {
                    input: {
                      id: img.id,
                      is_primary: false,
                    },
                  },
                });
              }
            }

            // Then set the selected image as primary
            await updateProfileImage({
              variables: {
                input: {
                  id: imageId,
                  is_primary: true,
                },
              },
            });

            // Update the local state - update profile_images with correct is_primary flags
            const updatedProfileImages =
              performer.profile_images?.map((img) => ({
                ...img,
                is_primary: img.id === imageId,
              })) || [];

            const primaryImage = updatedProfileImages.find(
              (img) => img.id === imageId
            );
            if (primaryImage) {
              onPerformerUpdate?.({
                ...performer,
                primary_image_path: primaryImage.image_path,
                profile_images: updatedProfileImages,
              });
            }

            Toast.success(
              intl.formatMessage(
                { id: "toast.updated_entity" },
                {
                  entity: intl
                    .formatMessage({ id: "performer" })
                    .toLocaleLowerCase(),
                }
              )
            );
          } catch (e) {
            Toast.error(e);
          }
        },
        [performer, onPerformerUpdate, updateProfileImage, intl, Toast]
      );

      const handleDeleteImage = useCallback(
        async (imageId: string, index: number) => {
          const confirmDelete = window.confirm(
            intl.formatMessage(
              {
                id: "dialogs.delete_confirm",
                defaultMessage: "Are you sure you want to delete {entityName}?",
              },
              { entityName: `Image ${index + 1}` }
            )
          );

          if (!confirmDelete) return;

          try {
            await destroyProfileImage({
              variables: {
                input: { id: imageId },
              },
            });

            Toast.success(
              intl.formatMessage(
                {
                  id: "toast.deleted_entity",
                  defaultMessage: "Deleted {entityType}",
                },
                { entityType: intl.formatMessage({ id: "image" }) }
              )
            );

            // Update local state - remove the deleted image
            const updatedProfileImages =
              performer.profile_images?.filter((img) => img.id !== imageId) ||
              [];
            onPerformerUpdate?.({
              ...performer,
              profile_images: updatedProfileImages,
            });

            // Adjust current index if needed
            if (
              currentImageIndex !== undefined &&
              index <= currentImageIndex &&
              currentImageIndex > 0
            ) {
              const newIndex = Math.max(0, currentImageIndex - 1);
              if (onImageChange) {
                onImageChange(newIndex);
              }
            }
          } catch (e) {
            Toast.error(e);
          }
        },
        [
          performer,
          onPerformerUpdate,
          destroyProfileImage,
          onImageChange,
          currentImageIndex,
          intl,
          Toast,
        ]
      );

      return (
        <HeaderImage hasImages={hasProfileImages}>
          <ProfileImageSlider
            profileImages={performer.profile_images || []}
            isEditing={isEditing}
            currentImageIndex={currentImageIndex}
            onImageChange={onImageChange}
            performerId={parseInt(performer.id, 10)}
            isNew={isNew}
            activeImage={activeImage}
            encodingImage={encodingImage}
            setImage={setImage}
            setEncodingImage={setEncodingImage}
            onPerformerUpdate={onPerformerUpdate}
            onSetPrimary={handleSetPrimary}
            onDeleteImage={handleDeleteImage}
            onImageUpdate={onImageUpdateProp}
            onAddImage={onAddImage}
          />
        </HeaderImage>
      );
    }
  );

const PerformerPage: React.FC<IProps> = PatchComponent(
  "PerformerPage",
  ({ performer, tabKey }) => {
    const Toast = useToast();
    const history = useHistory();
    const intl = useIntl();

    const [currentImageIndex, setCurrentImageIndex] = useState<number>(0);
    const currentImageIndexRef = useRef(currentImageIndex);

    // Keep ref in sync with state
    useEffect(() => {
      currentImageIndexRef.current = currentImageIndex;
    }, [currentImageIndex]);

    const onImageChange = useCallback((index: number) => {
      setCurrentImageIndex(index);
    }, []);

    // Configuration settings
    const { configuration } = React.useContext(ConfigurationContext);
    const uiConfig = configuration?.ui;
    const abbreviateCounter = uiConfig?.abbreviateCounters ?? false;
    const enableBackgroundImage =
      uiConfig?.enablePerformerBackgroundImage ?? false;
    const showAllDetails = uiConfig?.showAllDetails ?? true;
    const compactExpandedDetails = uiConfig?.compactExpandedDetails ?? false;

    const [collapsed, setCollapsed] = useState<boolean>(!showAllDetails);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const [image, setImage] = useState<string | null>();
    const [encodingImage, setEncodingImage] = useState<boolean>(false);
    const [currentPerformer, setCurrentPerformer] =
      useState<GQL.PerformerDataFragment>(performer);
    const loadStickyHeader = useLoadStickyHeader();

    // Refetch performer data hook
    const { refetch: refetchPerformer } = useFindPerformer(performer.id);

    // Update currentPerformer when performer prop changes
    const prevPerformerIdRef = useRef(performer.id);
    useEffect(() => {
      const isDifferentPerformer = prevPerformerIdRef.current !== performer.id;
      prevPerformerIdRef.current = performer.id;
      setCurrentPerformer(performer);
      if (isDifferentPerformer) {
        setCurrentImageIndex(0); // Reset to first image only when performer changes
      }
    }, [performer]);

    // Auto-switch to primary image when profileImages changes
    const prevProfileImagesRef = useRef(currentPerformer.profile_images);
    useEffect(() => {
      const profileImages = currentPerformer.profile_images;
      const prevProfileImages = prevProfileImagesRef.current;

      // Check if primary image actually changed
      const currentPrimary = profileImages?.find((img) => img.is_primary);
      const prevPrimary = prevProfileImages?.find((img) => img.is_primary);

      const primaryChanged = currentPrimary?.id !== prevPrimary?.id;

      if (profileImages && profileImages.length > 0 && primaryChanged) {
        const hasPrimaryImage = profileImages.some((img) => img.is_primary);
        if (hasPrimaryImage && currentImageIndexRef.current !== 0) {
          setCurrentImageIndex(0);
        }
      }

      prevProfileImagesRef.current = profileImages;
    }, [currentPerformer.profile_images]); // Only depend on profile_images to avoid loops

    const onImageUpdate = useCallback(async () => {
      try {
        // Refetch performer data to update the image after cropping
        const result = await refetchPerformer();
        if (result.data?.findPerformer) {
          setCurrentPerformer(result.data.findPerformer);
        }
      } catch (error) {
        console.error("Error refetching performer data:", error);
      }
    }, [refetchPerformer]);

    const onAddImageClick = useCallback(() => {
      // This will trigger the image input dialog
      console.log("Add image clicked");
    }, []);

    const onImageFileChange = useCallback(
      (event: React.ChangeEvent<HTMLInputElement>) => {
        // Handle the image file here
        const file = event.target.files?.[0];
        if (file) {
          console.log("New image selected:", file);
          // You can add logic to upload the image here
        }
      },
      []
    );

    const activeImage = useMemo(() => {
      const performerImage =
        currentPerformer.primary_image_path || currentPerformer.image_path;
      if (isEditing) {
        if (image === null && performerImage) {
          const performerImageURL = new URL(performerImage);
          performerImageURL.searchParams.set("default", "true");
          return performerImageURL.toString();
        } else if (image) {
          return image;
        }
      }
      return performerImage;
    }, [
      image,
      isEditing,
      currentPerformer.primary_image_path,
      currentPerformer.image_path,
    ]);

    const lightboxImages = useMemo(
      () => [{ paths: { thumbnail: activeImage, image: activeImage } }],
      [activeImage]
    );

    const [updatePerformer] = usePerformerUpdate();
    const [deletePerformer, { loading: isDestroying }] = usePerformerDestroy();

    async function onAutoTag() {
      try {
        await mutateMetadataAutoTag({ performers: [currentPerformer.id] });
        Toast.success(intl.formatMessage({ id: "toast.started_auto_tagging" }));
      } catch (e) {
        Toast.error(e);
      }
    }

    useRatingKeybinds(
      true,
      configuration?.ui.ratingSystemOptions?.type,
      setRating
    );

    // set up hotkeys
    useEffect(() => {
      Mousetrap.bind("e", () => toggleEditing());
      Mousetrap.bind("f", () => setFavorite(!currentPerformer.favorite));
      Mousetrap.bind(",", () => setCollapsed(!collapsed));

      return () => {
        Mousetrap.unbind("e");
        Mousetrap.unbind("f");
        Mousetrap.unbind(",");
      };
    });

    async function onSave(input: GQL.PerformerCreateInput) {
      await updatePerformer({
        variables: {
          input: {
            id: currentPerformer.id,
            ...input,
          },
        },
      });
      toggleEditing(false);
      Toast.success(
        intl.formatMessage(
          { id: "toast.updated_entity" },
          {
            entity: intl.formatMessage({ id: "performer" }).toLocaleLowerCase(),
          }
        )
      );
    }

    async function onDelete() {
      try {
        await deletePerformer({ variables: { id: performer.id } });
      } catch (e) {
        Toast.error(e);
        return;
      }

      history.goBack();
    }

    function toggleEditing(value?: boolean) {
      if (value !== undefined) {
        setIsEditing(value);
      } else {
        setIsEditing((e) => !e);
      }
      setImage(undefined);
    }

    function setFavorite(v: boolean) {
      if (performer.id) {
        updatePerformer({
          variables: {
            input: {
              id: currentPerformer.id,
              favorite: v,
            },
          },
        });
      }
    }

    function setRating(v: number | null) {
      if (performer.id) {
        updatePerformer({
          variables: {
            input: {
              id: currentPerformer.id,
              rating100: v,
            },
          },
        });
      }
    }

    const handleSave = useCallback(() => {
      // No-op function for save
    }, []);

    const handleImageChange = useCallback(() => {
      // No-op function for image change
    }, []);

    if (isDestroying)
      return (
        <LoadingIndicator
          message={`Deleting performer ${currentPerformer.id}: ${currentPerformer.name}`}
        />
      );

    const headerClassName = cx("detail-header", {
      edit: isEditing,
      collapsed,
      "full-width": !collapsed && !compactExpandedDetails,
    });

    return (
      <div id="performer-page" className="row">
        <Helmet>
          <title>{currentPerformer.name}</title>
        </Helmet>

        <div className={headerClassName}>
          <BackgroundImage
            imagePath={activeImage ?? undefined}
            show={enableBackgroundImage && !isEditing}
          />
          <div className="detail-container">
            <PerformerHeaderImage
              activeImage={activeImage}
              collapsed={collapsed}
              encodingImage={encodingImage}
              lightboxImages={lightboxImages}
              performer={currentPerformer}
              isEditing={isEditing}
              isNew={false}
              currentImageIndex={currentImageIndex}
              onImageChange={onImageChange}
              setImage={setImage}
              setEncodingImage={setEncodingImage}
              onPerformerUpdate={(updatedPerformer) =>
                setCurrentPerformer((prev) => ({
                  ...prev,
                  ...updatedPerformer,
                }))
              }
              onImageUpdate={onImageUpdate}
              onAddImage={onAddImageClick}
              onImageFileChange={onImageFileChange}
            />
            <div className="row">
              <div className="performer-head col">
                <DetailTitle
                  name={currentPerformer.name}
                  disambiguation={currentPerformer.disambiguation ?? undefined}
                  classNamePrefix="performer"
                >
                  {currentPerformer.primary_tag && (
                    <TagLink
                      tag={currentPerformer.primary_tag}
                      linkType="performer"
                      className="performer-primary-tag"
                    />
                  )}
                  {!isEditing && (
                    <ExpandCollapseButton
                      collapsed={collapsed}
                      setCollapsed={(v) => setCollapsed(v)}
                    />
                  )}
                  <span className="name-icons">
                    <FavoriteIcon
                      favorite={currentPerformer.favorite}
                      onToggleFavorite={(v) => setFavorite(v)}
                    />
                    <ExternalLinkButtons
                      urls={currentPerformer.urls ?? undefined}
                    />
                  </span>
                </DetailTitle>
                <RatingSystem
                  value={currentPerformer.rating100}
                  onSetRating={(value) => setRating(value)}
                  clickToRate
                  withoutContext
                />
                <AliasList aliases={currentPerformer.alias_list} />
                {!isEditing && (
                  <PerformerDetailsPanel
                    performer={performer}
                    collapsed={collapsed}
                    fullWidth={!collapsed && !compactExpandedDetails}
                  />
                )}
                {isEditing ? (
                  <PerformerEditPanel
                    performer={performer}
                    isVisible={isEditing}
                    onSubmit={onSave}
                    onCancel={() => toggleEditing()}
                    setImage={setImage}
                    setEncodingImage={setEncodingImage}
                    onPerformerUpdate={(updatedPerformer) =>
                      setCurrentPerformer((prev) => ({
                        ...prev,
                        ...updatedPerformer,
                      }))
                    }
                  />
                ) : (
                  <Col>
                    <Row xs={8}>
                      <DetailsEditNavbar
                        objectName={
                          performer?.name ??
                          intl.formatMessage({ id: "performer" })
                        }
                        onToggleEdit={() => toggleEditing()}
                        onDelete={onDelete}
                        onAutoTag={onAutoTag}
                        autoTagDisabled={currentPerformer.ignore_auto_tag}
                        isNew={false}
                        isEditing={false}
                        onSave={handleSave}
                        onImageChange={handleImageChange}
                        classNames="mb-2"
                        customButtons={
                          <div>
                            <PerformerSubmitButton performer={performer} />
                          </div>
                        }
                      ></DetailsEditNavbar>
                    </Row>
                  </Col>
                )}
              </div>
            </div>
          </div>
        </div>

        {!isEditing && loadStickyHeader && (
          <CompressedPerformerDetailsPanel performer={performer} />
        )}

        <div className="detail-body">
          <div className="performer-body">
            <div className="performer-tabs">
              {!isEditing && (
                <PerformerTabs
                  tabKey={tabKey}
                  performer={currentPerformer}
                  abbreviateCounter={abbreviateCounter}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }
);

const PerformerLoader: React.FC<RouteComponentProps<IPerformerParams>> = ({
  location,
  match,
}) => {
  const { id, tab } = match.params;
  const { data, loading, error } = useFindPerformer(id);

  useScrollToTopOnMount();

  if (loading) return <LoadingIndicator />;
  if (error) return <ErrorMessage error={error.message} />;
  if (!data?.findPerformer)
    return <ErrorMessage error={`No performer found with id ${id}.`} />;

  if (tab && !isTabKey(tab)) {
    return (
      <Redirect
        to={{
          ...location,
          pathname: `/performers/${id}`,
        }}
      />
    );
  }

  return (
    <PerformerPage
      performer={data.findPerformer}
      tabKey={tab as TabKey | undefined}
    />
  );
};

export default PerformerLoader;
