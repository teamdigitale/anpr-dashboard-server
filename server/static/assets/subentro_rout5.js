Date.prototype.isValid = function () {
  // An invalid date object returns NaN for getTime() and NaN is the only
  // object not strictly equal to itself.
  return this.getTime() === this.getTime();
};

   
var subentro = {};
subentro.searchFilter = {};

$(document).on('keydown', function(e) {
  if (e.keyCode === 9) {
    $('body').removeClass('hide-focus');
  } else if (e.keyCode == 191) { // slash
    $('#cerca').focus();
    e.preventDefault();
  }
})

$(document).on('click', function(e) {
  $('body').addClass('hide-focus');
})


var  subentro = {
    searchFilter: {},
    comuni: [],
    selectedComune: undefined,
    searchComuni: function(exp) {
      return subentro.comuni.filter(function(o) {
        return o.Name.toUpperCase().search(exp.toUpperCase()) > 0
      })
    },

    /*Return a date object in DD/MM/YYYY format */
    dateFormat: function(date) {
      return [date.getDate(), date.getMonth() + 1, date.getFullYear()].join("/");
    },

    /*Return a date object from a  DD/MM/YYYY date */
    getDateFromValue: function(value) {
      //r = value.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
      tokens = value.split("/");
      if(tokens.length=3){
          // month goes from 0 to 11 (yay) so we need the -1 below
          return new Date(Date.UTC(
              parseInt(tokens[2]), parseInt(tokens[1]) - 1, parseInt(tokens[0])))
      }
      return null;
    },

    calculateComuniWithSubentroDetails: function() {
      return subentro.comuni.filter(function(o) {
        return (o.Subentro.From && o.Subentro.To && !o.DataSubentro)
      })
    },

    calculateComuniAlreadySubscribed: function() {
      return subentro.comuni.filter(function(o) {
        return (o.DataSubentro)
      })
    },

    showComuniAggiornati: function() {
      $("#comuniAggiornati").html(
          "<p> Comuni Gestiti: " + subentro.comuni.length + "</p>" + 
          "<p> Con Data Pianificata: " + subentro.calculateComuniWithSubentroDetails().length + "</p>" + 
          "<p> Già Subentrati: " + subentro.calculateComuniAlreadySubscribed().length) + "</p>"

    },

    getComuneIndexById: function(id) {
      return subentro.comuni.findIndex(function(o) {
        return o.Id == id
      })
    },

    initAutoComplete: function() {
      var options = {
        data: subentro.comuni,
        getValue: function(e) {

          hasDate = e.Subentro.From ? " da:[" + subentro.dateFormat(new Date(e.Subentro.From)) + "]" : ""
          hasDate += e.Subentro.To ? " a:[" + subentro.dateFormat(new Date(e.Subentro.To)) + "]" : ""
          hasDate += e.Subentro.PreferredDate ? " data preferita:[" + subentro.dateFormat(new Date(e.Subentro.PreferredDate)) + "]" : ""
          hasDate += e.Subentro.IP ? " IP:[" + e.Subentro.IP + "]" : ""
 
          if(e.DataSubentro){
            return e.Name + " subentrato:" +subentro.dateFormat(new Date(e.DataSubentro))
          }else{
            return e.Name + hasDate;
          }
          
          
        },
        theme: "square",
        list: {
          match: {
            enabled: true
          },
          onClickEvent: function() {

            var selectedItemValue = $("#cercaComuni").getSelectedItemData();
            subentro.comune = selectedItemValue;


            $("#comuneSubentro").html(selectedItemValue.Name);
            var udpateDateIfNotEmpty = function(objectId, value) {
              if (value) {
                $("#" + objectId + "").val(subentro.dateFormat(new Date(value)))
              } else {
                $("#" + objectId + "").val("")
              }

            }
            udpateDateIfNotEmpty("interval-from", selectedItemValue.Subentro.From)
            udpateDateIfNotEmpty("interval-to", selectedItemValue.Subentro.To)
            udpateDateIfNotEmpty("interval-at", selectedItemValue.Subentro.PreferredDate)
            
            if(selectedItemValue.IPProvenienza){
              $("#ip").parent().hide();
            }else{
              $("#ip").parent().show();
            
            }
            if(selectedItemValue.Subentro.IP){
              $("#ip").val(selectedItemValue.Subentro.IP.String);

            }else{
              $("#ip").val("");
            }
           
            subentro.addComuneCheckList();
            comments.searchComments();//get comments for this particular
            $("#formSubentroComune").show();


          }
        },


      }
      $("#cercaComuni").easyAutocomplete(options);
    },

    addComuneCheckList: function(){
      console.log("Checklist for comune:", subentro.comune)
      $("#checkListForComune").show();
      if(subentro.comune.DataSubentro){
        $(".checkListComuneDati").html("<td colspan=\"10\">Comune subentrato il "+subentro.dateFormat(new Date(subentro.comune.DataSubentro))+"</td>");
      }else{
        
      var v =  "<td>";
      var abilitazionePrefettura = subentro.comune.AbilitazionePrefettura? "SI":"NO";
      v+=abilitazionePrefettura+"</td>";


      v+="</td>";
      v +=  "<td>";
      if (subentro.comune.DataAbilitazione){
        v += subentro.dateFormat(new Date(subentro.comune.DataAbilitazione));
      }
      v+="</td>";
      v +=  "<td>";
      if (subentro.comune.DataPresubentro){
        v += subentro.dateFormat(new Date(subentro.comune.DataPresubentro));
      }
      v+="</td>";
      v +=  "<td>";

      if (subentro.comune.DataConsegnaSm){
        v += subentro.dateFormat(new Date(subentro.comune.DataConsegnaSm));
      }
      v+="</td>";

      v +=  "<td>";

      if (subentro.comune.DataRitiroSm){
        v += subentro.dateFormat(new Date(subentro.comune.DataRitiroSm));
      }
      v+="</td>";
      v +=  "<td>";

      if (subentro.comune.NumeroLettori){
        v += subentro.comune.NumeroLettori;
      }
      v+="</td>";


      v +=  "<td>";

      if (subentro.comune.SCConsegnate){
        v += subentro.comune.SCConsegnate;
      }
      v+="</td>";

      v +=  "<td>";

      if (subentro.comune.Postazioni){
        v += subentro.comune.Postazioni;
      }
      v+="</td>";

      v +=  "<td>";

      if (subentro.comune.UtentiAbilitati){
        v += subentro.comune.UtentiAbilitati;
      }
      v+="</td>";

      v += "<td>";
      if (subentro.comune.DataPresubentro) {
        v += "<a href='/status?id="+subentro.comune.CodiceIstat+"' >"
        v += "<img src='static/assets/icons/gauge_icon.png' id='gauge' border=0>"
        v += "</a>"
      }
      v += "</td>";
      
    
      
      $(".checkListComuneDati").html(v);
      }
    }


  }
