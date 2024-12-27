package ecs

const (
	ENTITY_COMPONENT_MASK_ID ComponentID = 1<<8 - 1
	MAX_COMPONENTS_COUNT                 = COMPONENT_ID_RANGE_HI - COMPONENT_ID_RANGE_LO
	COMPONENT_ID_RANGE_LO                = 0
	COMPONENT_ID_RANGE_HI                = 254
)
