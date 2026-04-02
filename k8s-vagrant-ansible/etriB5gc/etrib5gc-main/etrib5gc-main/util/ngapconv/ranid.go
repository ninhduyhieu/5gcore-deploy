package ngapconv

import (
	//	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
)

func RanIdToModels(ranNodeId ies.GlobalRANNodeID) (ranId models.GlobalRanNodeId) {
	/*
		choice := ranNodeId.Choice
		switch choice {
		case ies.GlobalRANNodeIDPresentGlobalgnbId:
			ngapGnbId := ranNodeId.GlobalGNBID
			plmnid := PlmnIdToModels(ngapGnbId.PLMNIdentity)
			ranId.PlmnId = plmnid
			if ngapGnbId.GNBID.Choice == ies.GNBIDPresentGnbId {
				choiceGnbId := ngapGnbId.GNBID.GNBID
				ranId.GNbId = &models.GNbId{
					BitLength: int(choiceGnbId.NumBits),
					GNBValue:  BitStringToHex(choiceGnbId),
				}
			}
		case ies.GlobalRANNodeIDPresentGlobalngenbId:
			ngapNgENBID := ranNodeId.GlobalNgENBID
			plmnid := PlmnIdToModels(*ngapNgENBID.PLMNIdentity)
			ranId.PlmnId = plmnid
			if ngapNgENBID.NgENBID.Choice == ies.NgENBIDPresentMacroNgENBID {
				macroNgENBID := ngapNgENBID.NgENBID.MacroNgENBID
				ranId.NgeNbId = "MacroNGeNB-" + BitStringToHex(macroNgENBID)
			} else if ngapNgENBID.NgENBID.Choice == ies.NgENBIDPresentShortMacroNgENBID {
				shortMacroNgENBID := ngapNgENBID.NgENBID.ShortMacroNgENBID
				ranId.NgeNbId = "SMacroNGeNB-" + BitStringToHex(shortMacroNgENBID)
			} else if ngapNgENBID.NgENBID.Choice == ies.NgENBIDPresentLongMacroNgENBID {
				longMacroNgENBID := ngapNgENBID.NgENBID.LongMacroNgENBID
				ranId.NgeNbId = "LMacroNGeNB-" + BitStringToHex(longMacroNgENBID)
			}
		case ies.GlobalRANNodeIDPresentGlobaln3iwfId:
			ngapN3IWFID := ranNodeId.GlobalN3IWFID
			plmnid := PlmnIdToModels(*ngapN3IWFID.PLMNIdentity)
			ranId.PlmnId = plmnid
			if ngapN3IWFID.N3IWFID.Choice == ies.N3IWFIDPresentN3IWFID {
				choiceN3IWFID := ngapN3IWFID.N3IWFID.N3IWFID
				ranId.N3IwfId = BitStringToHex(choiceN3IWFID)
			}
		}
	*/
	return
}

func RanIDToNgap(mId models.GlobalRanNodeId) (ngapId ies.GlobalRANNodeID, err error) {
	/*
		if mId.GNbId != nil && mId.GNbId.BitLength != 0 {
			ngapId.Choice = ies.GlobalRANNodeIDPresentGlobalGNBID
			ngapId.GlobalGNBID = new(ies.GlobalGNBID)
			globalGNBID := ngapId.GlobalGNBID

			if globalGNBID.PLMNIdentity, err = PlmnIdToNgap(mId.PlmnId); err != nil {
				return
			}

			globalGNBID.GNBID.Choice = ies.GNBIDPresentGNBID
			globalGNBID.GNBID.GNBID = new(aper.BitString)
			if *globalGNBID.GNBID.GNBID, err = HexToBitString(mId.GNbId.GNBValue, int(mId.GNbId.BitLength)); err != nil {
				return
			}
		} else if mId.NgeNbId != "" {
			ngapId.Choice = ies.GlobalRANNodeIDPresentGlobalNgENBID
			ngapId.GlobalNgENBID = new(ies.GlobalNgENBID)
			globalNgENBID := ngapId.GlobalNgENBID

			if globalNgENBID.PLMNIdentity, err = PlmnIdToNgap(mId.PlmnId); err != nil {
				return
			}
			ngENBID := globalNgENBID.NgENBID
			if mId.NgeNbId[:11] == "MacroNGeNB-" {
				ngENBID.Choice = ies.NgENBIDPresentMacroNgENBID
				ngENBID.MacroNgENBID = new(aper.BitString)
				if *ngENBID.MacroNgENBID, err = HexToBitString(mId.NgeNbId[11:], 18); err != nil {
					return
				}
			} else if mId.NgeNbId[:12] == "SMacroNGeNB-" {
				ngENBID.Choice = ies.NgENBIDPresentShortMacroNgENBID
				ngENBID.ShortMacroNgENBID = new(aper.BitString)
				if *ngENBID.ShortMacroNgENBID, err = HexToBitString(mId.NgeNbId[12:], 20); err != nil {
					return
				}
			} else if mId.NgeNbId[:12] == "LMacroNGeNB-" {
				ngENBID.Choice = ies.NgENBIDPresentLongMacroNgENBID
				ngENBID.LongMacroNgENBID = new(aper.BitString)
				if *ngENBID.LongMacroNgENBID, err = HexToBitString(mId.NgeNbId[12:], 21); err != nil {
					return
				}
			}
		} else if mId.N3IwfId != "" {
			ngapId.Choice = ies.GlobalRANNodeIDPresentGlobalN3IWFID
			ngapId.GlobalN3IWFID = new(ies.GlobalN3IWFID)
			globalN3IWFID := ngapId.GlobalN3IWFID

			if globalN3IWFID.PLMNIdentity, err = PlmnIdToNgap(mId.PlmnId); err != nil {
				return
			}
			globalN3IWFID.N3IWFID.Choice = ies.N3IWFIDPresentN3IWFID
			globalN3IWFID.N3IWFID.N3IWFID = new(aper.BitString)
			if *globalN3IWFID.N3IWFID.N3IWFID, err = HexToBitString(mId.N3IwfId, len(mId.N3IwfId)*4); err != nil {
				return
			}
		}
	*/
	return
}
